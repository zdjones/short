package main

import (
	"net/http"
	"log"
	"time"

	"github.com/zdjones/short"
)


func main() {
	shortener, err := short.ShortenerHandler("http://lu.sh", "short.db")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/endpoint/", &shortener)
	
	s := &http.Server{
		Addr: 		":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Listening on %s...\n", s.Addr)
	log.Println(s.ListenAndServe())
}