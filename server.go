package main

import (
	"fmt"
	"log"
	"net/http"

	veni "../veni"
	vidi "../vidi"
)

func main() {
	http.HandleFunc("/call", veni.Handler)
	http.HandleFunc("/data", vidi.Handler)

	fmt.Printf("Starting server at port 3000\n")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
