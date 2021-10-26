package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

type requestSample struct {
	responseCode                 int
	timeStartTransfer, timeTotal time.Duration
}

func (r requestSample) isSuccessful() bool {
	if 199 < r.responseCode && r.responseCode < 300 {
		return true
	}
	return false
}

type probeResult struct {
	clients                                             int
	avgTimeStartTransferNano, avgTimeTotalNano, errRate float64
	responseCodeCount                                   map[int]int
}

func (p probeResult) String() string {
	return fmt.Sprintf("%d: avg(starttransfer)=%.2fms, avg(total)=%.2fms, error=%.1f%%", p.clients, p.avgTimeStartTransferNano/1000000, p.avgTimeTotalNano/1000000, p.errRate*100)
}

var (
	wg         sync.WaitGroup
	client     http.Client
	logEnabled bool
)

func main() {
	config := ParseConfig(os.Stderr)
	if config == nil {
		os.Exit(1)
	}

	if config.NoColor {
		color.NoColor = true
	}
	logEnabled = config.Verbose

	err := config.Request.Build()
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}

	color.Blue("GOMAXPROCS=%d", runtime.GOMAXPROCS(0))

	initClient(config.NumClients, config.Timeout, config.Insecure, &config.CaCert)

	process(config.Request, config.NumClients, config.NumRequests, config.Gradient)
}

func initClient(numClients int, timeout time.Duration, insecure bool, cacert *CaCert) {
	transport := http.DefaultTransport.(*http.Transport)
	transport.MaxConnsPerHost = numClients
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if cacert != nil && cacert.content!=nil {
		cacertPool := x509.NewCertPool()
		cacertPool.AppendCertsFromPEM(cacert.content)
		transport.TLSClientConfig = &tls.Config{RootCAs: cacertPool}
	}

	client = http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

func process(request Request, numClients, numRepeats int, accGradient float64) {
	color.Cyan("Connecting to %s...", request.URL)
	probes := make([]probeResult, 1, numClients+1)
	for i := 1; i <= numClients; i++ {
		probes = append(probes, exec(request, i, numRepeats))
		fmt.Print(probes[i])
		printGrad(&probes[i], &probes[i-1], accGradient)
		if i > 10 {
			printGrad(&probes[i], &probes[i-10], accGradient*10)
		}
		printResponseCodeCount(&probes[i])
		fmt.Println()
	}
}

func printGrad(current *probeResult, previous *probeResult, m float64) {
	if previous != nil && previous.avgTimeTotalNano != 0 {
		grad := current.avgTimeTotalNano / previous.avgTimeTotalNano
		dist := current.clients - previous.clients
		fmt.Printf(", grad(%d)=", -dist)
		switch {
		case grad > 2.0*m:
			color.Set(color.FgRed, color.Bold)
			break
		case grad > 1.6*m:
			color.Set(color.FgRed)
			break
		case grad > 1.2*m:
			color.Set(color.FgYellow)
			break
		case grad < 0.8*m:
			color.Set(color.FgGreen)
		}
		fmt.Printf("%.2f", grad)
		color.Unset()
	}
}

func printResponseCodeCount(current *probeResult) {
	color.Set(color.FgHiBlack)
	for code, count := range current.responseCodeCount {
		fmt.Printf(", rcc(%d)=%d", code, count)
	}
	color.Unset()
}

func exec(request Request, numClients, numRepeat int) probeResult {
	chanClientSample := make(chan []requestSample, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go doClientRequests(request, numRepeat, chanClientSample)
	}

	go func() {
		wg.Wait()
		close(chanClientSample)
	}()

	var sumTimStartTransfer, sumTimeTotal, successCount, errorCount int64
	codeCount := make(map[int]int)
	for clientSample := range chanClientSample {
		for i := 0; i < numRepeat; i++ {
			if clientSample[i].isSuccessful() {
				successCount++
				sumTimStartTransfer += clientSample[i].timeStartTransfer.Nanoseconds()
				sumTimeTotal += clientSample[i].timeTotal.Nanoseconds()
			} else {
				errorCount++
			}
			codeCount[clientSample[i].responseCode] = codeCount[clientSample[i].responseCode] + 1
		}
	}

	return probeResult{
		clients:                  numClients,
		avgTimeStartTransferNano: float64(sumTimStartTransfer) / float64(successCount),
		avgTimeTotalNano:         float64(sumTimeTotal) / float64(successCount),
		errRate:                  float64(errorCount) / float64(numClients*numRepeat),
		responseCodeCount:        codeCount,
	}
}

func doClientRequests(request Request, numRepeat int, chanClientSample chan<- []requestSample) {
	defer wg.Done()

	clientSample := make([]requestSample, numRepeat)
	for i := 0; i < numRepeat; i++ {
		clientSample[i] = *doRequest(request)
	}
	chanClientSample <- clientSample
}

func doRequest(request Request) *requestSample {

	result := requestSample{}

	req, _ := http.NewRequest(request.Method.String(), request.URL, bytes.NewBuffer(request.Body))
	for key, values := range request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	logRequest(req)

	start := time.Now()
	resp, err := client.Do(req)

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		fmt.Fprintf(os.Stderr, "timeout fetching: %v\n", err)
		return &result
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fetching failed: %v\n", err)
		return &result
	}
	defer resp.Body.Close()

	result.responseCode = resp.StatusCode
	result.timeStartTransfer = time.Now().Sub(start)

	body, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		fmt.Fprintf(os.Stderr, "reading failed: %v\n", bodyErr)
		return &result
	}

	result.timeTotal = time.Now().Sub(start)

	logReponse(resp, body)

	return &result
}

func logRequest(req *http.Request) {
	logVerbose("> " + req.Method + " " + req.URL.RequestURI() + " " + req.Proto)
	logVerbose("> Host: " + req.URL.Host)
	logVerboseHeader("> ", req.Header)
	logVerbose(">")
}

func logReponse(resp *http.Response, body []byte) {
	logVerbose("< " + resp.Proto + " " + resp.Status)
	logVerboseHeader("< ", resp.Header)
	logVerbose("<")
	logVerbose(string(body))
}

func logVerboseHeader(prefix string, header http.Header) {
	for key, values := range header {
		logVerbose(prefix + key + ": " + strings.Join(values, " "))
	}
}

func logVerbose(msg string) {
	if logEnabled {
		color.HiBlack(msg)
	}
}
