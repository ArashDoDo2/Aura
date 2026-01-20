package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/ArashDoDo2/Aura/internal"
)

func main() {
	// Configuration via flags or environment variables
	addr := flag.String("addr", getEnv("AURA_LISTEN_ADDR", ":53"), "Listen address (default :53)")
	domain := flag.String("domain", getEnv("AURA_DOMAIN", ""), "Authoritative domain (e.g., tunnel.example.com.)")
	targetHost := flag.String("target-host", getEnv("AURA_TARGET_HOST", internal.WhatsAppHost), "Target service host (default e1.whatsapp.net)")
	targetPort := flag.Int("target-port", getEnvInt("AURA_TARGET_PORT", internal.WhatsAppPort), "Target service port (default 5222)")
	flag.Parse()

	log.Printf("Starting Aura DNS Server")
	log.Printf("  Domain: %s", *domain)
	log.Printf("  Listen: %s", *addr)
	log.Printf("  Target: %s:%d", *targetHost, *targetPort)

	server := internal.NewServer(*domain, *targetHost, *targetPort)
	log.Fatal(server.ListenAndServe(*addr))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
		log.Printf("Invalid integer for %s: %q", key, value)
	}
	return defaultValue
}
