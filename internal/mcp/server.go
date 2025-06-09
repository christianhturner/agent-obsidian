package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/christianhturner/agent-obsidian/internal/obsidian"
)

type Server struct {
	name         string
	version      string
	capabilities ServerCapabilities
	stdin        io.Reader
	stdout       io.Writer
	initialized  bool

	quit chan os.Signal
}

func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		capabilities: ServerCapabilities{
			Resources: &ResourcesCapability{},
			Tools:     &ToolsCapability{},
		},
		quit: make(chan os.Signal, 1),
	}
}

func (s *Server) Run() error {
	return s.processInput()
	// scanner := bufio.NewScanner(s.stdin)
	//
	// for scanner.Scan() {
	// 	line := scanner.Bytes()
	// 	if len(line) == 0 {
	// 		continue
	// 	}
	//
	// 	if err := s.handleMessage(line); err != nil {
	// 		return fmt.Errorf("handling message: %w", err)
	// 	}
	// }
	//
	// return scanner.Err()
}

func (s *Server) processInput() error {
	scanner := bufio.NewScanner(s.stdin)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if err := s.handleMessage(line); err != nil {
			return fmt.Errorf("handling message: %w", err)
		}
	}

	return scanner.Err()
}

func (s *Server) Stop() {
	s.quit <- syscall.SIGTERM
}

func (s *Server) handleMessage(data []byte) error {
	// First, determine message type
	var base Message
	if err := json.Unmarshal(data, &base); err != nil {
		return s.sendError(-32700, "Parse error", nil)
	}

	// Handle request vs notification
	if base.ID != nil {
		// It's a request
		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			return s.sendError(-32700, "Parse error", *&base.ID)
		}
		return s.handleRequest(req)
	} else {
		// It's a notification (we'll implement later)
		return nil
	}
}

func (s *Server) handleRequest(req Request) error {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	default:
		return s.sendError(-32601, "Method not found", req.ID)
	}
}

func (s *Server) sendError(code int, message string, id *int) error {
	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: id},
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	return s.sendResponse(resp)
}

func (s *Server) sendResponse(resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(s.stdout, "%s\n", data)
	return err
}

func (s *Server) handleInitialize(req Request) error {
	var initReq InitializeRequest
	if err := json.Unmarshal(req.Params, &initReq); err != nil {
		return s.sendError(-32602, "Invalid params", req.ID)
	}

	// MCP protocol validation
	if initReq.ProtocolVersion != "2024-11-05" {
		return s.sendError(-32602, "Unsupported protocol version", req.ID)
	}

	// Build response
	initResp := InitializeResponse{
		ProtocolVersion: "2024-11-05",
		Capabilities:    s.capabilities,
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}

	result, err := json.Marshal(initResp)
	if err != nil {
		return s.sendError(-32603, "Internal error", req.ID)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: req.ID},
		Result:  result,
	}

	s.initialized = true
	return s.sendResponse(resp)
}

func (s *Server) handleListTools(req Request) error {
	if !s.initialized {
		return s.sendError(-32002, "Server not initialized", req.ID)
	}

	tools := []Tool{
		{
			Name:        "list_notes",
			Description: "List all notes in the Obsidian vault with metadata",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{},
			},
		},
		{
			Name:        "read_note",
			Description: "Read the full content of a specific note",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "The relative path of the note to read (e.g., 'First Note.md')",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "search_notes",
			Description: "Search notes by content, title, or tags",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Search query to find in note content or titles",
					},
					"tag": {
						Type:        "string",
						Description: "Tag to filter by (optional)",
					},
				},
				Required: []string{"query"},
			},
		},
	}

	response := ListToolsResponse{Tools: tools}
	result, err := json.Marshal(response)
	if err != nil {
		return s.sendError(-32603, "Internal error", req.ID)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: req.ID},
		Result:  result,
	}

	return s.sendResponse(resp)
}

func (s *Server) handleCallTool(req Request) error {
	if !s.initialized {
		return s.sendError(-32002, "Server not initialized", req.ID)
	}

	var callReq CallToolRequest
	if err := json.Unmarshal(req.Params, &callReq); err != nil {
		return s.sendError(-32602, "Invalid params", req.ID)
	}

	switch callReq.Name {
	case "list_notes":
		return s.handleListNotes(req.ID, callReq.Arguments)
	case "read_note":
		return s.handleReadNote(req.ID, callReq.Arguments)
	case "search_notes":
		return s.handleSearchNotes(req.ID, callReq.Arguments)
	default:
		return s.sendError(-32601, "Unknown tool", req.ID)
	}
}

func (s *Server) handleReadNote(id *int, args map[string]interface{}) error {
	// Extract path argument
	pathArg, ok := args["path"].(string)
	if !ok {
		return s.sendToolError(id, "Missing or invalid 'path' argument")
	}

	// Validate path safety
	if strings.Contains(pathArg, "..") || strings.HasPrefix(pathArg, "/") {
		return s.sendToolError(id, "Invalid path: path traversal not allowed")
	}

	// Load config
	config, err := obsidian.LoadConfig("config.json")
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to load config: %v", err))
	}

	// Build and validate full path
	fullPath := filepath.Join(config.VaultPath, pathArg)

	// Ensure the resolved path is still within vault
	absVaultPath, err := filepath.Abs(config.VaultPath)
	if err != nil {
		return s.sendToolError(id, "Failed to resolve vault path")
	}

	absNotePath, err := filepath.Abs(fullPath)
	if err != nil {
		return s.sendToolError(id, "Failed to resolve note path")
	}

	if !strings.HasPrefix(absNotePath, absVaultPath) {
		return s.sendToolError(id, "Path outside vault not allowed")
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to read note: %v", err))
	}

	response := CallToolResponse{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(content),
			},
		},
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendError(-32603, "Internal error", id)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: id},
		Result:  result,
	}

	return s.sendResponse(resp)
}

func (s *Server) handleListNotes(id *int, args map[string]interface{}) error {
	// Load config and create vault
	config, err := obsidian.LoadConfig("config.json")
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to load config: %v", err))
	}

	vault, err := obsidian.NewVault(config)
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to create vault: %v", err))
	}

	// List notes
	notes, err := vault.ListNotes()
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to list notes: %v", err))
	}

	// Format response
	notesJSON, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return s.sendToolError(id, "Failed to format notes")
	}

	response := CallToolResponse{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(notesJSON),
			},
		},
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendError(-32603, "Internal error", id)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: id},
		Result:  result,
	}

	return s.sendResponse(resp)
}

func (s *Server) sendToolError(id *int, message string) error {
	response := CallToolResponse{
		Content: []ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Error: %s", message),
			},
		},
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendError(-32603, "Internal error", id)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: id},
		Result:  result,
	}

	return s.sendResponse(resp)
}

func (s *Server) handleSearchNotes(id *int, args map[string]interface{}) error {
	// Extract search arguments
	query, ok := args["query"].(string)
	if !ok {
		return s.sendToolError(id, "Missing or invalid 'query' argument")
	}

	// Optional tag filter
	tagFilter, _ := args["tag"].(string)

	// Load config and create vault
	config, err := obsidian.LoadConfig("config.json")
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to load config: %v", err))
	}

	vault, err := obsidian.NewVault(config)
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to create vault: %v", err))
	}

	// Search notes
	results, err := vault.SearchNotes(query, tagFilter)
	if err != nil {
		return s.sendToolError(id, fmt.Sprintf("Failed to search notes: %v", err))
	}

	// Format response
	resultsJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return s.sendToolError(id, "Failed to format search results")
	}

	response := CallToolResponse{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(resultsJSON),
			},
		},
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendError(-32603, "Internal error", id)
	}

	resp := Response{
		Message: Message{JSONRPC: "2.0", ID: id},
		Result:  result,
	}

	return s.sendResponse(resp)
}
