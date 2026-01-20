package main

import (
	"flag"
	"log"
	"os"

	"github.com/ArashDoDo2/Aura/internal"
)

func main() {
	// Configuration via flags or environment variables
	addr := flag.String("addr", getEnv("AURA_LISTEN_ADDR", ":53"), "Listen address (default :53)")
	domain := flag.String("domain", getEnv("AURA_DOMAIN", "aura.net."), "Authoritative domain (e.g., aura.net.)")
	flag.Parse()

	log.Printf("Starting Aura DNS Server")
	log.Printf("  Domain: %s", *domain)
	log.Printf("  Listen: %s", *addr)

	server := internal.NewServer(*domain)
	log.Fatal(server.ListenAndServe(*addr))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
