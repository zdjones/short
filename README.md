short is an HTTP Handler that provides the backend and API for a URL Shortener. 
Add this Handler to a route on your server, and you have your own URL Shortener.

To shorten a URL, make a POST request with the long URL alone in the request body.
The response should contain the short URL in the response body

To Expand a URL, make a GET request with the short url encoded with the key 'short'.
The server should redirect to the expanded long URL.

```
go run examples/simpleserver/main.go
2019/04/26 19:37:56 Listening on :8080...
```
From a second terminal window:
```
go run short-cli/main.go http://localhost:8080/endpoint http://long.com/abcdefghijklmnopqrstuvwxyz
2019/04/26 19:38:34 Attempting to shorten http://long.com/abcdefghijklmnopqrstuvwxyz via http://localhost:8080/endpoint
http://lu.sh/1

go run short-cli/main.go -x http://localhost:8080/endpoint http://lu.sh/1
2019/04/26 19:38:40 Attempting to Expand http://lu.sh/1 via http://localhost:8080/endpoint
http://long.com/abcdefghijklmnopqrstuvwxyz
```