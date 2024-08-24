package main

import (
	"fmt"
	"log"
	"net/http"

	veni "github.com/Emperor42/veni"
	vidi "github.com/Emperor42/vidi"
)

func main() {
	http.Handle("/call", &veni.VeniContext{Name: "veni"})
	http.Handle("/data", &vidi.VidiContext{Name: "vidi"})

	fmt.Printf("Starting server at port 3000\n")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
