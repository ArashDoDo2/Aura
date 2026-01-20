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
	dnsServer := flag.String("dns", "8.8.8.8:53", "DNS server address")
	domain := flag.String("domain", "aura.net.", "Domain root for Aura")
	flag.Parse()

	err := internal.StartAuraClient(*dnsServer, *domain)
	if err != nil {
		fmt.Println("Failed to start Aura client:", err)
		os.Exit(1)
	}
	fmt.Println("Aura client started on port 1080. Press Ctrl+C to stop.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nStopping Aura client...")
	internal.StopAuraClient()
	time.Sleep(1 * time.Second)
	fmt.Println("Stopped.")
}
