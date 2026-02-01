package main

import (
	"flag"
	"fmt"
	"log"
)

const (
	Version = "0.0.1"
)

func main() {
	var (
		showVersion bool
		command     string
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		command = args[0]
	}

	if showVersion {
		fmt.Printf("V6Coin Node v%s\n", Version)
		return
	}

	switch command {
	case "start":
		fmt.Println("Starting V6Coin node...")
		// TODO: Implement node start logic
	case "stop":
		fmt.Println("Stopping V6Coin node...")
		// TODO: Implement node stop logic
	case "status":
		fmt.Println("V6Coin node status:")
		// TODO: Implement node status check
	default:
		log.Fatal("Unknown command. Use: start|stop|status")
	}
}
