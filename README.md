# cHail

Simulates parallel access to URLs through a configurable number of clients

[![Build Status](https://app.travis-ci.com/mcjr/chail.svg?branch=master)](https://app.travis-ci.com/mcjr/chail)

## Usage

        Usage: chail [options...]> <url>
        -h, --help                       This help text
        --no-color                       No color output
        -v, --verbose                    Make the operation more talkative
        --clients int                    Number of clients (default 1)
        --iterations int                 Number of sucessive requests for every client (default 1)
        --gradient float                 Accepted gradient of expected linear function (default 1.1)
        --connect-timeout duration       Maximum time allowed for connection (default 1s)
        -k, --insecure                   TLS connections without certs
        --cacert file                    CA certificate file (PEM)
        -X, --request command            Request command to use (GET, POST) (default GET)
        -H, --header header              Custom http header data
        -d, --data data/@file            Post data; filenames are prefixed with @
        -F, --form name=content          Multipart POST data; filenames are prefixed with @, e.g. <name>=@<path/to/file>;type=<override content-type>

## Example

    chail --clients 1 --iterations 1 \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer 243545" \
        -d '{"info": "Updated"}'  \
        http://localhost:8000/product/123

sends the request

        POST /product/123 HTTP/1.1
        Header["User-Agent"] = ["chail"]
        Header["Content-Length"] = ["19"]
        Header["Accept"] = ["*/*"]
        Header["Accept-Encoding"] = ["gzip"]
        Header["Authorization"] = ["Bearer 243545"]
        Header["Content-Type"] = ["application/json"]

and could produce the following output:

        Connecting to http://localhost:8000/product/123...
        1: avg(starttransfer)=0.63ms, avg(total)=0.65ms, error=0.0%, rcc(200)=1

## Build from sources

Setup a workspace as described in https://golang.org/doc/code.html.

        git clone https://github.com/mcjr/chail.git
        cd chail
        go build

### Running from sources

        go run . [options...] <url>

### Testing

Run test verbosely:

        go test -v 

Run test with coverage analysis:

        go test -coverprofile cover.out
        go tool cover -html=cover.out -o cover.html

## Future plans

* Add median
