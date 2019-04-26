package main

import (
	"log"
	"net/http"

	"github.com/zdjones/short"
)

func main() {
	shortener, err := short.ShortenerHandler("http://lu.sh", "short.db")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/endpoint", &shortener)

	log.Printf("Listening on %s...\n", ":8080")
	log.Println(http.ListenAndServe(":8080", nil))
}
