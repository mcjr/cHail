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

type probeResult struct {
	clients int
	avgNano float64
	errRate float64
}

func (p probeResult) String() string {
	return fmt.Sprintf("%d: avg=%.2f ms, err=%.1f", p.clients, p.avgNano/1000000, p.errRate)
}

var (
	wg         sync.WaitGroup
	client     http.Client
	logEnabled bool
)

func main() {
	config := ParseConfig()
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
	if cacert != nil {
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
		fmt.Println()
	}
}

func printGrad(current *probeResult, previous *probeResult, m float64) {
	if previous != nil && previous.avgNano != 0 {
		grad := current.avgNano / previous.avgNano
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

func exec(request Request, numClients, numRepeat int) probeResult {
	durations := make(chan time.Duration, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go probeRequests(request, numRepeat, durations)
	}

	go func() {
		wg.Wait()
		close(durations)
	}()

	var sum, cnt int64
	for duration := range durations {
		sum += duration.Nanoseconds()
		cnt++
	}

	return probeResult{
		clients: numClients,
		avgNano: float64(sum) / float64(cnt),
		errRate: 100.0 - 100.0*float64(cnt)/float64(numClients*numRepeat),
	}
}

func probeRequests(request Request, numRepeat int, durations chan<- time.Duration) {
	defer wg.Done()

	for i := 0; i < numRepeat; i++ {
		start := time.Now()

		ok := doRequest(request)

		elapsed := time.Now().Sub(start)

		if ok {
			durations <- elapsed
		}
	}
}

func doRequest(request Request) bool {

	req, _ := http.NewRequest(request.Method.String(), request.URL, bytes.NewBuffer(request.Body))
	for key, values := range request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	logRequest(req)

	resp, err := client.Do(req)

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		fmt.Fprintf(os.Stderr, "timeout fetching: %v\n", err)
		return false
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fetching failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	body, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		fmt.Fprintf(os.Stderr, "reading failed: %v\n", bodyErr)
		return false
	}

	logReponse(resp, body)

	return 200 <= resp.StatusCode && resp.StatusCode < 300
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

func logVerboseHeader(prefix string, header http.Header ) {
	for key, values := range header {
		logVerbose(prefix + key + ": " + strings.Join(values, " "))
	}
}

func logVerbose(msg string) {
	if logEnabled {
		color.HiBlack(msg)
	}
}
