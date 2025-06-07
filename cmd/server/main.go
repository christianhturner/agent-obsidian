package main

import (
	"log"

	"github.com/christianhturner/agent-obsidian/internal/mcp"
)

func main() {
	server := mcp.NewServer("agent-obsidian", "0.1.0")

	log.Println("Agent obsidian MCP Server starting...")
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
