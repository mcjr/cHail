package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
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
	wg sync.WaitGroup

	client    http.Client
	reqMethod Method
	reqHeader = make(Header)
	reqData   Data
)

func main() {
	numClients := flag.Int("clients", 20, "Number of clients")
	numRequests := flag.Int("iterations", 5, "Number of sucessive requests for every client")
	accGradient := flag.Float64("gradient", 1.1, "Accepted gradient of expected linear function")
	conTimeout := flag.Duration("connect-timeout", time.Duration(1*time.Second), "Maximum time allowed for connection")
	noColor := flag.Bool("no-color", false, "No color output")

	flagVarWithMultipleNames(&reqMethod, "Request command to use (GET, POST)", "X", "command")
	flagVarWithMultipleNames(&reqHeader, "Custom http header line", "H", "header")
	flagVarWithMultipleNames(&reqData, "Post data; filenames are prefixed with @", "d", "data")

	flag.Parse()
	urls := flag.Args()

	if len(urls) < 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: chail [options...]> <url>...\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *noColor {
		color.NoColor = true
	}

	color.Blue("GOMAXPROCS=%d", runtime.GOMAXPROCS(0))

	client = http.Client{
		Timeout: *conTimeout,
	}

	for _, link := range urls {
		color.Cyan("Connecting to %s...", link)
		probes := make([]probeResult, 1, *numClients+1)
		for i := 1; i <= *numClients; i++ {
			probes = append(probes, exec(link, i, *numRequests))
			fmt.Print(probes[i])
			printGrad(&probes[i], &probes[i-1], *accGradient)
			if i > 10 {
				printGrad(&probes[i], &probes[i-10], *accGradient*10)
			}
			fmt.Println()
		}
	}
}

func flagVarWithMultipleNames(value flag.Value, usage string, names ...string) {
	for _, name := range names {
		flag.Var(value, name, usage)
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

func exec(link string, numClients, numRepeat int) probeResult {
	durations := make(chan time.Duration, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go probeRequests(link, numRepeat, durations)
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

func probeRequests(url string, numRepeat int, durations chan<- time.Duration) {
	defer wg.Done()

	for i := 0; i < numRepeat; i++ {
		start := time.Now()

		ok := doRequest(url)

		elapsed := time.Now().Sub(start)

		if ok {
			durations <- elapsed
		}
	}
}

func doRequest(url string) bool {

	req, _ := http.NewRequest(reqMethod.String(), url, bytes.NewBuffer(reqData.content))
	for key, values := range reqHeader {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		fmt.Fprintf(os.Stderr, "timeout fetching: %v\n", err)
		return false
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fetching failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading failed: %v\n", err)
		return false
	}

	return 200 <= resp.StatusCode && resp.StatusCode < 300
}
