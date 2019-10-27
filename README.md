# cHail

Simulates parallel access to URLs through a configurable number of clients

[![Build Status](https://travis-ci.org/mcjr/chail.svg?branch=master)](https://travis-ci.org/mcjr/chail)

## Usage

        Usage: chail [options...]> <url>
        Options:
        -F value
                Multipart POST data; filenames are prefixed with @, e.g. <name>=@<path/to/file>;type=<override content-type>
        -H value
                Custom http header line
        -X value
                Request command to use (GET, POST)
        -cacert value
                CA certificate file (PEM)
        -clients int
                Number of clients (default 20)
        -command value
                Request command to use (GET, POST)
        -connect-timeout duration
                Maximum time allowed for connection (default 1s)
        -d value
                Post data; filenames are prefixed with @
        -data value
                Post data; filenames are prefixed with @
        -form value
                Multipart POST data; filenames are prefixed with @, e.g. <name>=@<path/to/file>;type=<override content-type>
        -gradient float
                Accepted gradient of expected linear function (default 1.1)
        -h    This help text
        -header value
                Custom http header line
        -help
                This help text
        -insecure
                TLS connections without certs
        -iterations int
                Number of sucessive requests for every client (default 5)
        -k    TLS connections without certs
        -no-color
                No color output

Each option can also be provided with a two dash prefix.

## Example

    chail -clients 1 -iterations 1 \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer 243545" \
        -d '{"info": "Updated"}'  \
        http://localhost:8000/product/123

sends the following request:

        POST /product/123 HTTP/1.1
        Header["Authorization"] = ["Bearer 243545"]
        Header["Content-Type"] = ["application/json"]
        Header["Accept-Encoding"] = ["gzip"]
        Header["User-Agent"] = ["Go-http-client/1.1"]
        Header["Content-Length"] = ["19"]

## Build from sources

Setup a workspace as described in https://golang.org/doc/code.html.

        cd $GOPATH/src
        git clone https://github.com/mcjr/chail.git
        cd chail
        go build

### Running from sources

        go run chail.go flags.go [options...] <url>

### Testing

Run test verbosely:

        go test -v 

Run test with coverage analysis:

        go test -coverprofile cover.out
        go tool cover -html=cover.out -o cover.html

## Future plans

* Add median
* Add verbose option