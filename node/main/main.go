package main

import (
	"log"
	"os"
	"strconv"

	core "sbimain"
)

func main() {
	// Загрузка этического хеша
	ethHash := os.Getenv("ISOTOPE_ETHICS_HASH")
	if ethHash == "" {
		// Fallback: попробовать загрузить из файла (для локальной разработки)
		ethData, err := os.ReadFile("genesis/ethics_hash.txt")
		if err != nil {
			log.Fatal("ISOTOPE_ETHICS_HASH not set and genesis/ethics_hash.txt not found. Set the environment variable or create the file.")
		}
		ethHash = core.HashText(string(ethData))
		log.Println("Ethics hash loaded from genesis/ethics_hash.txt, hash:", ethHash)
	} else {
		ethHash = core.HashText(ethHash)
		log.Println("Ethics hash loaded from ISOTOPE_ETHICS_HASH environment variable, hash:", ethHash)
	}

	// Определяем порт из ENV
	port := 9000
	if portStr := os.Getenv("ISOTOPE_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// HTTP-порт из ENV
	httpPort := 8081
	if portStr := os.Getenv("ISOTOPE_HTTP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			httpPort = p
		}
	}

	// Создаём конфигурацию
	cfg := core.Config{
		EthHash:    ethHash,
		Transports: []string{"ws", "tcp"},
		Port:       port,
		EnableMDNS: true,
	}

	// Создаём узел
	node := core.NewNode(cfg)

	// Инициализируем P2P
	if err := node.InitP2P(); err != nil {
		log.Fatal("Failed to init P2P:", err)
	}

	// Запускаем HTTP-сервер
	go func() {
		if err := node.StartHTTP(httpPort); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Блокируем
	select {}
}