# cHail

Simulates parallel access to URLs through a configurable number of clients

## Usage

    Usage: chail [options...]> <url>...
    Options:
    -H value
            header
    -clients int
            number of clients (default 20)
    -connect-timeout duration
            Maximum time allowed for connection (default 1s)
    -gradient float
            accepted gradient of expected linear function (default 1.1)
    -iterations int
            number of sucessive requests for every client (default 5)
    -no-color
            No color output

## Example

    chail -clients 30 -header "Content-Type: application/json" -header "Authorization: Bearer 243545"  http://localhost:8000/list http://localhost:8080/greeting 

## Future plans

* Support POST
* Add verbose option
