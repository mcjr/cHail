package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExec(t *testing.T) {
	setUp("GET", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startServer(t, "Content-Type", "application/json")
	defer server.Close()

	probe := exec(server.URL, 2, 2)
	if probe.clients != 2 || probe.errRate > 0.0 {
		t.Errorf("exec fails, expected %d clients %f error rate, but was %d clients and %f error rate!", 2, 0.0, probe.clients, probe.errRate)
	}
}

func TestDoRequestGET(t *testing.T) {
	setUp("GET", "Content-Type: application/xml", `<xml><entry key="1" value="2"/></xml>`)
	server := startServer(t, "Content-Type", "application/xml")
	defer server.Close()

	ok := doRequest(server.URL)
	if !ok {
		t.Errorf("doRequest fails: %s %s", reqMethod.String(), server.URL)
	}
}

func TestDoRequestPOST(t *testing.T) {
	setUp("GET", "Content-Type: application/json", `{"key1":"value1", "key2":"value2"}`)
	server := startServer(t, "Content-Type", "application/json")
	defer server.Close()

	ok := doRequest(server.URL)
	if !ok {
		t.Errorf("doRequest fails: %s %s", reqMethod.String(), server.URL)
	}
}

func setUp(method, headerLine, data string) {
	reqMethod.Set(method)
	reqHeader = make(Header)
	reqHeader.Set(headerLine)
	reqData.Set(data)
}

func startServer(t *testing.T, key, value string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != reqMethod.String() {
			t.Errorf("Request has method %s, but expected %s", r.Method, reqMethod.String())
			http.Error(w, "invalid method", http.StatusBadRequest)
			return
		}
		if r.Header[key][0] != value {
			t.Errorf("Expected header %q with value %q but got %v!", key, value, r.Header)
			http.Error(w, "invalid header", http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Error reading request body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		if string(body) != reqData.String() {
			t.Errorf("Expected body %s, but was %s", string(body), reqData.String())
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
	}))
}
