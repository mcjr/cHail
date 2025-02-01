# cHail

Simulates parallel access to URLs through a configurable number of clients

[![Build Status](https://github.com/mcjr/chail/actions/workflows/go.yml/badge.svg)](https://github.com/mcjr/chail/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/mcjr/chail)](https://goreportcard.com/report/github.com/mcjr/chail)

## Usage

        Usage: chail [options...]> <url>
        -h, --help                       This help text
        --no-color                       No color output
        -v, --verbose                    Make the operation more talkative
        --compressed                     Send header 'Accept-Encoding' with values 'deflate', 'gzip'
        --clients int                    Number of clients (default 1)
        --repeats int                    Number of successive requests for every client (default 1)
        --gradient float                 Accepted gradient of expected linear function (default 1.1)
        --connect-timeout duration       Maximum time allowed for connection (default 1s)
        -k, --insecure                   TLS connections without certs
        --cacert file                    CA certificate file (PEM)
        -X, --request command            Request command to use (GET, POST) (default GET)
        -H, --header header              Custom http header data
        -d, --data data/@file            Post data; filenames are prefixed with @
        -F, --form name=content          Multipart POST data; filenames are prefixed with @, e.g. <name>=@<path/to/file>;type=<override content-type>

## Example

Executing

        chail --clients 20 --repeats 5 \
              -H "Content-Type: application/json" \
              -H "Authorization: Bearer 243545" \
              -d @example.json \
              http://localhost:8000/product/123

simulates from 1 to 20 parallel clients, where each client executes 5 requests sequentially. Each request is in the form

        POST /product/123 HTTP/1.1
        Header["User-Agent"] = ["chail"]
        Header["Content-Length"] = ["19783"]
        Header["Accept"] = ["*/*"]
        Header["Accept-Encoding"] = ["gzip"]
        Header["Authorization"] = ["Bearer 243545"]
        Header["Content-Type"] = ["application/json"]

The above execution then could produce the following output:

![output](/output.png "example output")

Worth mentioning are the specifications for the functions _grad_ and _rcc_:

   * _grad(offset)_ is the abbreviation for gradient and sets the current average value of the total time in relation to the corresponding value of the specified _offset_. For example, grad(-1) calculates the quotient of the current value and the previous value. There is a fixed representation with regard to gradient values:

      * grad < 0.8: green
      * grad > 1.2: yellow
      * grad > 1.6: red
      * grad > 2.0: red, bold
   
   * _rcc(value)_ indicates the number of times _value_ occurs as a response code.

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
