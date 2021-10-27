package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

func TestParseConfigEmpty(t *testing.T) {
	var buf bytes.Buffer
	flag.CommandLine = flag.NewFlagSet("Empty", flag.PanicOnError)
	flag.CommandLine.SetOutput(io.Writer(&buf))
	os.Args = []string{"chail"}
	config := ParseConfig(io.Writer(&buf))
	if config != nil {
		t.Errorf("Missing arguments not recognized!")
	}
	if buf.String() != "Missing URL!\n" {
		t.Errorf("Error message is not printed, got %q!", buf.String())
	}
}

func TestParseConfigHelp(t *testing.T) {
	var buf bytes.Buffer
	flag.CommandLine = flag.NewFlagSet("Help", flag.PanicOnError)
	//	flag.CommandLine.SetOutput(io.Writer(&buf))
	os.Args = []string{"chail", "-h"}
	config := ParseConfig(io.Writer(&buf))
	if config != nil {
		t.Errorf("Missing arguments not recognized!")
	}
	if !strings.HasPrefix(buf.String(), "Usage: chail [options...]> <url>") {
		t.Errorf("Usage is not printed!")
	}
}

func TestParseConfigInsecure(t *testing.T) {
	var buf bytes.Buffer
	flag.CommandLine = flag.NewFlagSet("Insecure", flag.PanicOnError)
	os.Args = []string{"chail",
		"-k",
		"http://localhost:8080"}
	c := ParseConfig(io.Writer(&buf))
	assertConfigSecure(t, c, true, "")
	assertConfigRequest(t, c, GET, "http://localhost:8080", "", "", "")
}

func TestParseConfigStandard(t *testing.T) {
	var buf bytes.Buffer
	flag.CommandLine = flag.NewFlagSet("Standard", flag.PanicOnError)
	os.Args = []string{"chail",
		"--no-color",
		"--clients", "4",
		"--iterations", "5",
		"--gradient", "1.2",
		"--connect-timeout", "2s",
		"-k",
		"--cacert", "flags_test.pem",
		"-X", "POST",
		"-H", "Content-Encoding: UTF-8",
		"-d", "key=value",
		"http://localhost:8080"}
	c := ParseConfig(io.Writer(&buf))
	if !c.NoColor {
		t.Errorf("Option 'no-color' not recognized!")
	}
	if c.Timeout.String() != "2s" {
		t.Errorf("Invalid value for option 'Timeout': %q (expected %q)", c.Timeout.String(), "2s")
	}
	assertConfigCommon(t, c, 4, 5, 1.2)
	assertConfigSecure(t, c, true, "flags_test.pem")
	assertConfigRequest(t, c, POST, "http://localhost:8080", "map[Content-Encoding: UTF-8]", "key=value", "")
}

func TestParseConfigMultiPartForm(t *testing.T) {
	var buf bytes.Buffer

	flag.CommandLine = flag.NewFlagSet("MultiPartForm", flag.PanicOnError)
	os.Args = []string{"chail",
		"--clients=4",
		"--iterations=5",
		"--gradient=1.2",
		"--form", "dat=@flags_test.json;type=application/json",
		"--form", "val=data",
		"http://localhost:8080"}
	c := ParseConfig(io.Writer(&buf))
	assertConfigCommon(t, c, 4, 5, 1.2)
	assertConfigRequest(t, c, POST, "http://localhost:8080", "", "", "#Value=1, #File=1")
}

func assertConfigCommon(t *testing.T, c *Config, expectedClients, expectedIteractions int, expectedGradient float64) {
	if c.NumClients != expectedClients {
		t.Errorf("Invalid value for option 'Number of clients': %d (expected %d)", c.NumClients, expectedClients)
	}
	if c.NumRequests != expectedIteractions {
		t.Errorf("Invalid value for option 'Number of requests': %d (expected %d)", c.NumRequests, expectedIteractions)
	}
	if c.Gradient != expectedGradient {
		t.Errorf("Invalid value for option 'Gradient': %f (expected %f)", c.Gradient, expectedGradient)
	}
}

func assertConfigSecure(t *testing.T, c *Config, expectedInsecure bool, expectedCaCertPath string) {
	if c.Insecure != expectedInsecure {
		t.Errorf("Invalid value for option 'Insecure': %t (expected %t)", c.Insecure, expectedInsecure)
	}
	if expectedCaCertPath != "" {
		expectedCaCert, _ := ioutil.ReadFile(expectedCaCertPath)
		if c.CaCert.String() != string(expectedCaCert) {
			t.Errorf("Invalid value for option 'CaCert': %q (expected %q)", c.CaCert.String(), string(expectedCaCert))
		}
	}
}

func assertConfigRequest(t *testing.T, c *Config, expectedMethod Method, expectedURL, expectedHeader, expectedData, expectedMultiPartFormData string) {
	if c.Request.Method != expectedMethod {
		t.Errorf("Invalid value for option 'Request command': %q (expected %q)", c.Request.Method.String(), expectedMethod.String())
	}
	if c.Request.URL != expectedURL {
		t.Errorf("Invalid value for 'URL': %q (expected %q)", c.Request.URL, expectedURL)
	}
	if c.Request.Header.String() != expectedHeader {
		t.Errorf("Invalid value for option 'Header': %q (expected %q)", c.Request.Header.String(), expectedHeader)
	}
	if c.Request.Data.String() != expectedData {
		t.Errorf("Invalid value for option 'Data': %q (expected %q)", c.Request.Data.String(), expectedData)
	}
	if c.Request.MultiPartFormData.String() != expectedMultiPartFormData {
		t.Errorf("Invalid value for option 'MultiPartForm': %q (expected %q)", c.Request.MultiPartFormData.String(), expectedMultiPartFormData)
	}
}

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
	line := "@not-exists.json"
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

func TestCaCertSet(t *testing.T) {
	filename := "./flags_test.pem"
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Setup test failed! %s %v", filename, err)
		return
	}
	assertCaCert(t, filename, string(content))
}

func TestCaCertSetWithError(t *testing.T) {
	var cacert CaCert
	filename := "./missing.pem"
	err := cacert.Set(filename)
	if err == nil {
		t.Errorf("CaCert.Set(%q) must return an error, because of missing file!", filename)
	}
}

func assertCaCert(t *testing.T, filename, expectedCaCert string) {
	var cacert CaCert
	err := cacert.Set(filename)
	if err != nil {
		t.Errorf("CaCert.Set(%q) has error: %v", filename, err)
		return
	}
	if cacert.String() != expectedCaCert {
		t.Errorf("CaCert.Set(%q) has invalid value: %q, expected %q", filename, cacert.String(), expectedCaCert)
	}
}

func TestMultiPartFormDataSet(t *testing.T) {
	assertValueOfMultiPartFormDataSet(t, "name", "", "")
	assertValueOfMultiPartFormDataSet(t, "name={key='value'}", "name", "{key='value'}")
	assertFileOfMultiPartFormDataSet(t, "name=@path/to/file", "name", "path/to/file", "")
	assertFileOfMultiPartFormDataSet(t, "name=@path/to/file;type=application/json", "name", "path/to/file", "application/json")
	assertFileOfMultiPartFormDataSet(t, "name=@path/to/file;type=application/json;more", "name", "path/to/file", "application/json;more")
	assertFileOfMultiPartFormDataSet(t, "name=@path/to/file;invalid=application/json", "",  "", "")
}

func assertValueOfMultiPartFormDataSet(t *testing.T, arg, expectedName, expectedValue string) {
	m := NewMultiPartFormData()
	err := m.Set(arg)
	if err != nil && (expectedName != "" || expectedValue != "") {
		t.Errorf("MultiPartFormData.Set(%q) has error: %v", arg, err)
		return
	}
	if expectedValue != "" {
		if len(m.Value[expectedName]) != 1 {
			t.Errorf("MultiPartFormData.Set(%q) causes no unique value!", arg)
		}
		if m.Value[expectedName][0] != expectedValue {
			t.Errorf("MultiPartFormData.Set(%q) has invalid value: %q, expected %q", arg, m.File[expectedName][0].Filename, expectedValue)
		}
		if !strings.HasPrefix(m.String(), "#Value=1") {
			t.Errorf("MultiPartFormData.String() has invalid value: %q", m.String())
		}
	}

}

func assertFileOfMultiPartFormDataSet(t *testing.T, arg, expectedName, expectedFile, expectedOverrideType string) {
	m := NewMultiPartFormData()
	err := m.Set(arg)
	if err != nil && (expectedName != "" || expectedFile != "" || expectedOverrideType != "") {
		t.Errorf("MultiPartFormData.Set(%q) has error: %v", arg, err)
		return
	}
	if expectedFile != "" {
		if len(m.File[expectedName]) != 1 {
			t.Errorf("MultiPartFormData.Set(%q) causes no unique value!", arg)
		}
		if m.File[expectedName][0].Filename != expectedFile {
			t.Errorf("MultiPartFormData.Set(%q) has invalid value: %q, expected %q", arg, m.File[expectedName][0].Filename, expectedFile)
		}
		if len(m.File[expectedName][0].Header) != 2 {
			t.Errorf("MultiPartFormData.Set(%q) has missing file type value!", arg)
		}
		if len(m.File[expectedName][0].Header["Content-Disposition"]) < 1 {
			t.Errorf("MultiPartFormData.Set(%q) has missing content disposition", arg)
		}
		if expectedOverrideType != "" &&  expectedOverrideType != m.File[expectedName][0].Header["Content-Type"][0] {
			t.Errorf("MultiPartFormData.Set(%q) has missing file type value: %q, expected %q", arg, m.File[expectedName][0].Header["Content-Type"], expectedOverrideType)
		}		
		if !strings.HasSuffix(m.String(), "#File=1") {
			t.Errorf("MultiPartFormData.String() has invalid value: %q", m.String())
		}
	}
}

func TestParseProperty(t *testing.T) {
	assertParseProperty(t, "", "", "")
	assertParseProperty(t, "a", "", "")
	assertParseProperty(t, "a=", "a", "")
	assertParseProperty(t, "a==", "a", "=")
	assertParseProperty(t, "a=b", "a", "b")
	assertParseProperty(t, "a=b=", "a", "b=")
	assertParseProperty(t, " a =   ", "a", "")
	assertParseProperty(t, " a = b ", "a", "b")
}

func assertParseProperty(t *testing.T, arg, expectedName, expectedValue string) {
	name, value := parseProperty(arg)
	if name != expectedName || value != expectedValue {
		t.Errorf("Assertion for parseProperty(%q) fails, got name %q (expecting %q) and value %q (expecting %q)", arg, name, expectedValue, value, expectedValue)
	}
}
