package main

import (
	"net/http"
	"testing"
)

func TestHeaderSet(t *testing.T) {
	assertHeaderSet(t, "Content-Type: application/json", "Content-Type", []string{"application/json"})
	assertHeaderSet(t, "content-type: application/json", "Content-Type", []string{"application/json"})
	assertHeaderSet(t, "Content-Type: application/x-www-form-urlencoded", "Content-Type", []string{"application/x-www-form-urlencoded"})

	assertHeaderSetWithMultipleLines(t,
		[]string{"Content-Type: application/json", "Content-Type: application/xml"},
		"Content-Type",
		[]string{"application/json", "application/xml"})
}

func TestHeaderSetWithError(t *testing.T) {
	line := "Content-Type=application/json"
	header := make(Header)
	err := header.Set(line)
	if err == nil {
		t.Errorf("Header.Set(%q) must return an error: %v", line, err)
	}
}

func assertHeaderSet(t *testing.T, line, expectedKey string, expectedValues []string) {
	assertHeaderSetWithMultipleLines(t, []string{line}, expectedKey, expectedValues)
}

func assertHeaderSetWithMultipleLines(t *testing.T, lines []string, expectedKey string, expectedValues []string) {
	header := make(Header)
	for _, line := range lines {
		err := header.Set(line)
		if err != nil {
			t.Errorf("Header.Set(%q) has error: %v", line, err)
			return
		}
	}
	if len(header) != 1 {
		t.Errorf("Header.Set(%q) failed, missing entry: %v", lines, len(header))
		return
	}

	values := header[expectedKey]
	if len(values) != len(expectedValues) {
		t.Errorf("Header.Set(%q) failed, different values: %d <> %d", lines, len(values), len(expectedValues))
		return
	}

	for _, expectedValue := range expectedValues {
		if !contains(values, expectedValue) {
			t.Errorf("Header.Set(%q) failed, missing value: %v", lines, expectedValue)
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func TestMethodSet(t *testing.T) {
	assertMethod(t, "GET", http.MethodGet)
	assertMethod(t, "POST", http.MethodPost)
}

func TestMethodSetWithError(t *testing.T) {
	line := "PUT"
	var method Method
	err := method.Set(line)
	if err == nil {
		t.Errorf("Method.Set(%q) should return error!", line)
	}
}

func assertMethod(t *testing.T, line, expectedMethod string) {
	var method Method
	err := method.Set(line)
	if err != nil {
		t.Errorf("Method.Set(%q) has error: %v", line, err)
		return
	}
	if method.String() != expectedMethod {
		t.Errorf("Method.Set(%q) results in an invalid value: %q, expected %q", line, method.String(), expectedMethod)
	}
}

func TestDataSet(t *testing.T) {
	assertData(t, "", "")
	line := `{"info": "Updated"}`
	assertData(t, line, line)
	assertData(t, "@./flags_test.json", line)
}

func TestDataSetWithError(t *testing.T) {
	var data Data
	line := "@not-existis.json"
	err := data.Set(line)
	if err == nil {
		t.Errorf("Data.Set(%q) must return an error, because of missing file!", line)
	}
}

func assertData(t *testing.T, line, expectedData string) {
	var data Data
	err := data.Set(line)
	if err != nil {
		t.Errorf("Data.Set(%q) has error: %v", line, err)
		return
	}
	if data.String() != expectedData {
		t.Errorf("Data.Set(%q) has invalid value: %q, expected %q", line, data.String(), expectedData)
	}
}
