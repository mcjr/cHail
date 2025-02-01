package main

import (
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	config Config
)

func TestProcess(t *testing.T) {
	setUp("GET", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startServer(t, "Content-Type", "application/json")
	defer server.Close()

	process(config.Request, 11, 1, 1.1)
}

func TestProcessWithErrors(t *testing.T) {
	setUp("GET", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startResponseCodeServer(429)
	defer server.Close()

	process(config.Request, 1, 1, 1.1)
}

func TestExec(t *testing.T) {
	setUp("GET", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startServer(t, "Content-Type", "application/json")
	defer server.Close()

	probe := exec(config.Request, 2, 2)
	if probe.clients != 2 || probe.errRate > 0.0 {
		t.Errorf("exec fails, expected %d clients %f error rate, but was %d clients and %f error rate!", 2, 0.0, probe.clients, probe.errRate)
	}
}

func TestDoRequestTLS(t *testing.T) {
	setUp("GET", "Content-Type: application/xml", `<xml><entry key="1" value="2"/></xml>`)

	server := startTLSServer(t, "Content-Type", "application/xml")
	defer server.Close()

	if !strings.HasPrefix(server.URL, "https:") {
		t.Errorf("Expected protocol %q, but URL is %q", "https:", server.URL)
	}

	certContent := server.TLS.Certificates[0].Certificate[0]
	pemContent := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certContent})
	cacert := CaCert{pemContent}

	initClient(1, time.Duration(1*time.Second), false, &cacert)

	sample := doRequest(config.Request)
	if !sample.isSuccessful() {
		t.Errorf("doRequest fails: %s %s", config.Request.Method.String(), server.URL)
	}
}

func TestDoRequestInsecureTLS(t *testing.T) {
	setUp("GET", "Content-Type: application/xml", `<xml><entry key="1" value="2"/></xml>`)

	server := startTLSServer(t, "Content-Type", "application/xml")
	defer server.Close()

	if !strings.HasPrefix(server.URL, "https:") {
		t.Errorf("Expected protocol %q, but URL is %q", "https:", server.URL)
	}

	initClient(1, time.Duration(1*time.Second), true, nil)

	sample := doRequest(config.Request)
	if !sample.isSuccessful() {
		t.Errorf("doRequest fails: %s %s", config.Request.Method.String(), server.URL)
	}
}

func TestDoRequestGET(t *testing.T) {
	setUp("GET", "Content-Type: application/xml", `<xml><entry key="1" value="2"/></xml>`)
	server := startServer(t, "Content-Type", "application/xml")
	defer server.Close()

	sample := doRequest(config.Request)
	if !sample.isSuccessful() {
		t.Errorf("doRequest fails: %s %s", config.Request.Method.String(), server.URL)
	}
}

func TestDoRequestGETButClientError(t *testing.T) {
	setUp("GET", "Content-Type: application/xml", `<xml><entry key="1" value="2"/></xml>`)
	server := startResponseCodeServer(400)
	defer server.Close()

	sample := doRequest(config.Request)
	if sample.isSuccessful() {
		t.Errorf("doRequest should fail with response code 400: %d %d", 400, sample.responseCode)
	}
}

func TestDoRequestPOST(t *testing.T) {
	setUp("POST", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startServer(t, "Content-Type", "application/json")
	defer server.Close()

	sample := doRequest(config.Request)
	if !sample.isSuccessful() {
		t.Errorf("doRequest fails: %s %s", config.Request.Method.String(), server.URL)
	}
}

func setUp(method, headerLine, data string) {
	config.Request.Method.Set(method)
	config.Request.Header = make(Header)
	config.Request.Header.Set(headerLine)
	config.Request.Data.Set(data)
	config.Request.Build()
}

func startResponseCodeServer(responseCode int) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(responseCode) }))
	config.Request.URL = ts.URL
	return ts
}

func startServer(t *testing.T, key, value string) *httptest.Server {
	th := TestHandler{t, key, value}
	ts := httptest.NewServer(http.HandlerFunc(th.handle))
	config.Request.URL = ts.URL
	return ts
}

func startTLSServer(t *testing.T, key, value string) *httptest.Server {
	th := TestHandler{t, key, value}
	ts := httptest.NewTLSServer(http.HandlerFunc(th.handle))
	config.Request.URL = ts.URL
	return ts
}

type TestHandler struct {
	*testing.T
	key, value string
}

func (t *TestHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != config.Request.Method.String() {
		t.Errorf("Request has method %s, but expected %s", r.Method, config.Request.Method.String())
		http.Error(w, "invalid method", http.StatusBadRequest)
		return
	}
	if r.Header[t.key][0] != t.value {
		t.Errorf("Expected header %q with value %q but got %v!", t.key, t.value, r.Header)
		http.Error(w, "invalid header", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Error reading request body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	if string(body) != config.Request.Data.String() {
		t.Errorf("Expected body %s, but was %s", string(body), config.Request.Data.String())
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
}
func (t *TestHandler) handleError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "custom error", http.StatusBadRequest)
}
