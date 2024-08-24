package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Emperor42/veni"
	"github.com/Emperor42/vidi"
)

func main() {
	http.Handle("/call", &veni.VeniContext{"veni"})
	http.Handle("/data", &vidi.VidiContext{"veni"})

	fmt.Printf("Starting server at port 3000\n")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
