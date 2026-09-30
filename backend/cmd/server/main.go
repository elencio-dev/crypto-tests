package main

import (
	"log"
	"net/http"

	"crypto-lab/internal/algorithms/ecdh"
	"crypto-lab/internal/engine"
	"crypto-lab/internal/httpapi"
)

func main() {
	registry := engine.NewRegistry()
	registry.Register(ecdh.New())

	server := httpapi.NewServer(registry)

	const addr = ":8081"
	log.Printf("crypto-lab backend listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
