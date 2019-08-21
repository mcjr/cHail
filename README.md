# cHail

Simulates parallel access to URLs through a configurable number of clients

## Usage

    Usage: cHail [options...]> <url> [<url>]*
    Options:
    -clients int
            number of clients (default 20)
    -connect-timeout duration
            Maximum time allowed for connection (default 1s)
    -gradient float
            accepted gradient of expected linear function (default 0.1)
    -iterations int
            number of sucessive requests for every client (default 5)

## Example

    cHail -clients 30  http://localhost:8000/list http://localhost:8080/greeting 

## Future plans

* Support request header
* Support POST
