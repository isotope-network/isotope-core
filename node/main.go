package main

import (
	"log"
	"os"
)

func main() {
	ethHash := os.Getenv("ISOTOPE_ETHICS_HASH")
	if ethHash == "" {
		// Fallback: попробовать загрузить из файла (для локальной разработки)
		ethData, err := os.ReadFile("genesis/ethics_hash.txt")
		if err != nil {
			log.Fatal("ISOTOPE_ETHICS_HASH not set and genesis/ethics_hash.txt not found. Set the environment variable or create the file.")
		}
		ethHash = hashText(string(ethData))
		log.Println("Ethics hash loaded from genesis/ethics_hash.txt, hash:", ethHash)
	} else {
		ethHash = hashText(ethHash)
		log.Println("Ethics hash loaded from ISOTOPE_ETHICS_HASH environment variable, hash:", ethHash)
	}

	node := &Node{
		ethHash: ethHash,
		memory: Memory{
			seen: make(map[string]bool),
		},
	}

	node.start()
}