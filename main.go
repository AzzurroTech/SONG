package main

import (
	"fmt"
	"log"
	"net/http"

	veni "../veni"
	vidi "../vidi"
)

func main() {
	http.Handle("/call", &veni.VeniContext{"veni"})
	http.Handle("/data", &vidi.VidiContext{"veni"})

	fmt.Printf("Starting server at port 3000\n")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
