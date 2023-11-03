package main

import (
	"context"
	"github.com/lukaspj/go-masterserver/pkg/httpserver"
	"github.com/lukaspj/go-masterserver/pkg/lobby"
	"github.com/lukaspj/go-masterserver/pkg/tcp"
	"log"
)

func main() {
	service := lobby.NewService()
	closeChan := make(chan error)
	ctx := context.Background()
	go func(closeChan chan<- error) {
		server := httpserver.NewServer(service)
		err := server.ListenAndServe()
		closeChan <- err
	}(closeChan)
	go func(closeChan chan<- error) {
		server := tcp.NewServer(service)
		err := server.ListenAndServe(ctx)
		closeChan <- err
	}(closeChan)

	for i := 0; i < 2; i++ {
		err := <-closeChan
		if err != nil {
			log.Fatal(err)
		}
	}
}
