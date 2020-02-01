package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"reflect"
	"strings"
	"time"
)

// Config is build from flags and arguments
type Config struct {
	NoColor, Insecure       bool
	NumClients, NumRequests int
	Gradient                float64
	Timeout                 time.Duration
	Request                 Request
	CaCert                  CaCert
}

func newConfig() *Config {
	return &Config{
		Request: Request{
			Header:            Header{},
			MultiPartFormData: *NewMultiPartFormData(),
		},
	}
}

// ParseConfig from command line
func ParseConfig() *Config {
	c := newConfig()

	help := false
	flagBoolVar(&help, false, "This help text", "h", "help")

	flag.BoolVar(&c.NoColor, "no-color", false, "No color output")

	flag.IntVar(&c.NumClients, "clients", 1, "Number of clients")
	flag.IntVar(&c.NumRequests, "iterations", 1, "Number of sucessive requests for every client")
	flag.Float64Var(&c.Gradient, "gradient", 1.1, "Accepted gradient of expected linear function")

	flag.DurationVar(&c.Timeout, "connect-timeout", time.Duration(1*time.Second), "Maximum time allowed for connection")

	flagBoolVar(&c.Insecure, false, "TLS connections without certs", "k", "insecure")
	flagVar(&c.CaCert, "CA certificate file (PEM)", "cacert")

	flagVar(&c.Request.Method, "Request command to use (GET, POST)", "X", "command")
	flagVar(&c.Request.Header, "Custom http header line", "H", "header")
	flagVar(&c.Request.Data, "Post data; filenames are prefixed with @", "d", "data")
	flagVar(&c.Request.MultiPartFormData, "Multipart POST data; filenames are prefixed with @, e.g. <name>=@<path/to/file>;type=<override content-type>", "F", "form")

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if help {
		usage()
		return nil
	}

	if len(args) != 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Missing URL!\n")
		return nil
	}
	c.Request.URL = args[0]

	if !c.Request.Data.IsEmpty() && !c.Request.MultiPartFormData.IsEmpty() {
		fmt.Fprintf(flag.CommandLine.Output(), "Can not use data and multi part form data in a request!\n")
		return nil
	}

	if !c.Request.Data.IsEmpty() || !c.Request.MultiPartFormData.IsEmpty() {
		c.Request.Method = POST
	}

	return c
}

func flagBoolVar(value *bool, initValue bool, usage string, names ...string) *bool {
	for _, name := range names {
		flag.BoolVar(value, name, initValue, usage)
	}
	return value
}

func flagVar(value flag.Value, usage string, names ...string) {
	for _, name := range names {
		flag.Var(value, name, usage)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: chail [options...]> <url>\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")

	usageOrder := []flag.Flag{}
	usageFlags := make(map[string][]flag.Flag)
	flag.VisitAll(func(f *flag.Flag) {
		usageOrder = append(usageOrder, *f)
		usageFlags[f.Usage] = append(usageFlags[f.Usage], *f)
	})
	for _, f := range usageOrder {
		fl := usageFlags[f.Usage]
		if fl != nil {
			s := "  "
			f1 := fl[0]
			for i := range fl {
				s += fmt.Sprintf("-%s", fl[i].Name)
				name, _ := flag.UnquoteUsage(&fl[i])
				if len(name) > 0 {
					s += " " + name
				}
				if i < len(fl)-1 {
					s += ", "
				}
			}
			s += "\n    \t"
			s += strings.ReplaceAll(f1.Usage, "\n", "\n    \t")
			if !isZeroValue(&f1, f1.DefValue) {
				s += fmt.Sprintf(" (default %v)", f1.DefValue)
			}
			fmt.Fprint(flag.CommandLine.Output(), s, "\n")

			usageFlags[f.Usage] = nil
		}
	}
}

// flag.isZeroValue is unfortunately internal
func isZeroValue(f *flag.Flag, value string) bool {
	typ := reflect.TypeOf(f.Value)
	return value == reflect.New(typ.Elem()).Interface().(flag.Value).String()
}

// Request from arguments
type Request struct {
	Method            Method
	URL               string
	Header            Header
	Data              Data
	MultiPartFormData MultiPartFormData
	Body			  []byte
}

// Build Request after config is parsed
func (r *Request) Build() error {
	if !r.Data.IsEmpty() {
		r.Body = r.Data.content
	}

	if !r.MultiPartFormData.IsEmpty() {
		content := new(bytes.Buffer)
		writer := multipart.NewWriter(content)
	
		for _, fileHeaders := range r.MultiPartFormData.File {
			for _, fileHeader := range fileHeaders {
				file, err := os.Open(fileHeader.Filename)
				if err != nil {
					return err
				}
				fileContents, err := ioutil.ReadAll(file)
				if err != nil {
					return err
				}
				file.Close()
	
				part, err := writer.CreatePart(fileHeader.Header)
				if err != nil {
					return err
				}
				part.Write(fileContents)
			}
		}
		for key, values := range r.MultiPartFormData.Value {
			for _, value := range values {
				_ = writer.WriteField(key, value)
			}
		}
		err := writer.Close()
		if err != nil {
			return fmt.Errorf("unable to close content: %q", err)
		}
		r.Body = content.Bytes()
	}

	return nil
}

// Header from arguments
type Header http.Header

func (h Header) String() string {
	s := "map["
	for k, v := range h {
		s += fmt.Sprintf("%s: %s", k, strings.Join(v, " "))
	}
	return s + "]"
}

// Set Header f rom arguments
func (h Header) Set(s string) error {
	key, value := parse2Terms(s, ":")
	if key != "" {
		mimeHeaderkey := textproto.CanonicalMIMEHeaderKey(key)
		h[mimeHeaderkey] = append(h[mimeHeaderkey], value)
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

// IsEmpty is true if and only if content is empty
func (d *Data) IsEmpty() bool {
	return len(d.content) == 0
}

// MultiPartFormData from arguments
type MultiPartFormData multipart.Form

// NewMultiPartFormData is the construcor
func NewMultiPartFormData() *MultiPartFormData {
	return &MultiPartFormData{
		Value: map[string][]string{},
		File:  map[string][]*multipart.FileHeader{},
	}
}

func (m *MultiPartFormData) String() string {
	return fmt.Sprintf("#Value=%d, #File=%d", len(m.Value), len(m.File))
}

// Set MultiPartFormData from argument
func (m *MultiPartFormData) Set(s string) error {
	// <name>=@<path/to/file>;type=<override content-type>
	parts := strings.SplitN(s, ";", 2)
	if len(parts) > 0 {
		name, value := parseProperty(parts[0])
		if name != "" {
			if strings.HasPrefix(value, "@") {
				fh := new(multipart.FileHeader)
				fh.Filename = strings.TrimPrefix(value, "@")
				fh.Header = make(textproto.MIMEHeader)
				fh.Header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(name), escapeQuotes(fh.Filename)))
				fh.Header.Set("Content-Type", "application/octet-stream")
				if len(parts) > 1 {
					key, overridenType := parseProperty(parts[1])
					if strings.ToLower(key) == "type" {
						fh.Header.Set("Content-Type", overridenType)
					} else {
						return fmt.Errorf("invalid file type in multi part form data string %q", s)
					}
				}
				m.File[name] = append(m.File[name], fh)
			} else {
				m.Value[name] = append(m.Value[name], value)
			}
		} else {
			return fmt.Errorf("invalid multi part form data string %q", s)
		}
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// IsEmpty is true if and only if no file and no value exists
func (m *MultiPartFormData) IsEmpty() bool {
	return len(m.Value) == 0 && len(m.File) == 0
}

func parseProperty(s string) (string, string) {
	return parse2Terms(s, "=")
}

func parse2Terms(s, sep string) (string, string) {
	terms := strings.SplitN(s, sep, 2)
	if len(terms) == 2 {
		return strings.TrimSpace(terms[0]), strings.TrimSpace(terms[1])
	}
	return "", ""
}

// CaCert from arguments
type CaCert struct {
	content []byte
}

func (c *CaCert) String() string {
	return string(c.content)
}

// Set CaCert from argument
func (c *CaCert) Set(s string) error {
	var err error
	c.content, err = ioutil.ReadFile(s)
	if err != nil {
		return err
	}
	return nil
}
