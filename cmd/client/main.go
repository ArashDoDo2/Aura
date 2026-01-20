package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArashDoDo2/Aura/internal"
)

func main() {
	dnsServer := flag.String("dns", getEnv("AURA_DNS_SERVER", ""), "DNS server address (empty = use system resolver)")
	domain := flag.String("domain", getEnv("AURA_DOMAIN", ""), "Target domain (e.g., tunnel.example.com.)")
	port := flag.Int("port", getEnvInt("AURA_SOCKS5_PORT", 1080), "SOCKS5 proxy port")
	flag.Parse()

	err := internal.StartAuraClient(*dnsServer, *domain, *port)
	if err != nil {
		fmt.Println("Failed to start Aura client:", err)
		os.Exit(1)
	}
	fmt.Printf("Aura client started on port %d. Press Ctrl+C to stop.\n", *port)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nStopping Aura client...")
	internal.StopAuraClient()
	time.Sleep(1 * time.Second)
	fmt.Println("Stopped.")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}
