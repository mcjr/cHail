package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
)

// Header from arguments
type Header http.Header

func (h Header) String() string {
	return fmt.Sprintf("#%T=%d", h, len(h))
}

// Set Header from argument
func (h Header) Set(s string) error {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		key := textproto.CanonicalMIMEHeaderKey(parts[0])
		h[key] = append(h[key], strings.TrimSpace(parts[1]))
		return nil
	}
	return fmt.Errorf("invalid header string %q", s)
}

// Method from arguments
type Method int

const (
	// GET method
	GET Method = iota
	// POST method
	POST
)

func (m *Method) String() string {
	switch *m {
	case GET:
		return http.MethodGet
	case POST:
		return http.MethodPost
	}
	return ""
}

// Set Method from argument
func (m *Method) Set(s string) error {
	switch s {
	case "GET":
		*m = GET
		return nil
	case "POST":
		*m = POST
		return nil
	}
	return fmt.Errorf("invalid method string %q", s)
}

// Data from arguments
type Data struct {
	content []byte
}

func (d *Data) String() string {
	return string(d.content)
}

// Set Data from argument
func (d *Data) Set(s string) error {
	if strings.HasPrefix(s, "@") {
		var err error
		d.content, err = ioutil.ReadFile(strings.TrimPrefix(s, "@"))
		if err != nil {
			return err
		}
	} else {
		d.content = []byte(s)
	}
	return nil
}
