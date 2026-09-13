package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"strconv"

	core "sbimain"
)

func main() {
	ethHash := os.Getenv("ISOTOPE_ETHICS_HASH")
	if ethHash == "" {
		ethData, err := os.ReadFile("commandments.txt")
		if err != nil {
			log.Fatal("ISOTOPE_ETHICS_HASH not set and commandments.txt not found. Set the environment variable or create the file.")
		}
		sum := sha256.Sum256(ethData)
		ethHash = hex.EncodeToString(sum[:])
		log.Println("Ethics hash loaded from commandments.txt, hash:", ethHash)
	} else {
		log.Println("Ethics hash loaded from ISOTOPE_ETHICS_HASH environment variable, hash:", ethHash)
	}

	port := 9000
	if portStr := os.Getenv("ISOTOPE_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	httpPort := 8081
	if portStr := os.Getenv("ISOTOPE_HTTP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			httpPort = p
		}
	}

	enableRelayServer := false
	if relayStr := os.Getenv("ISOTOPE_ENABLE_RELAY"); relayStr == "true" {
		enableRelayServer = true
	}

	cfg := core.Config{
		EthHash:           ethHash,
		Transports:        []string{"ws", "tcp"},
		Port:              port,
		EnableMDNS:        true,
		EnableRelayServer: enableRelayServer,
	}

	node := core.NewNode(cfg)

	if err := node.InitP2P(); err != nil {
		log.Fatal("Failed to init P2P:", err)
	}

	go func() {
		if err := node.StartHTTP(httpPort); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	select {}
}