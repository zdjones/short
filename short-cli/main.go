package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {

	expand := flag.Bool("x", false, "Expand a short URL")
	// long := flag.String("l", flag.Args[1], "long url")
	// short := flag.String("s", flag.Args[1], "short url")
	// addr := flag.String("api", flag.Args[2], "API endpoint")
	flag.Parse()

	if len(flag.Args()) < 2 {
		log.Fatalln("describe use here")
	}

	api, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatalf("Invalid URL (%s): %s\n", flag.Arg(0), err)
	}
	u, err := url.Parse(flag.Arg(1))
	if err != nil {
		log.Fatalf("Invalid URL (%s): %s", flag.Arg(1), err)
	}

	// Don't follow redirects, we only need the location header.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if *expand {
		log.Printf("Attempting to Expand %s via %s\n", u.String(), api.String())
		q := api.Query()
		q.Set("short", u.String())
		api.RawQuery = q.Encode()
		resp, err := client.Get(api.String())
		if err != nil {
			log.Fatalf("Error calling api %s: %s\n", api.String(), err)
		}

		//check status?
		location, err := resp.Location()
		if err != nil {
			log.Fatalf("Expand error response(%d): %s\n", resp.StatusCode, err)
		}
		fmt.Println(location.String())
		os.Exit(0)
	}

	log.Printf("Attempting to shorten %s via %s\n", u.String(), api.String())
	resp, err := client.Post(api.String(), "text/plain", strings.NewReader(u.String()))
	if err != nil {
		log.Fatalf("Error calling api %s: %s\n", api.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		long, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading repsonse body: %s\n", err)
		}
		fmt.Println(string(long))
	}

}
