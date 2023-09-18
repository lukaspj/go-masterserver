package main

import (
	"github.com/lukaspj/go-masterserver/pkg/httpserver"
	"log"
)

func main() {
	server := httpserver.NewServer()
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
