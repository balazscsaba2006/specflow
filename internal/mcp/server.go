package mcp

import (
	"context"
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/config"
	sfcontext "github.com/balazscsaba2006/specflow/internal/context"
	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server with specflow-specific dependencies.
type Server struct {
	store   *store.Store
	cfg     config.Config
	builder *sfcontext.Builder
	mcpSrv  *mcp.Server
}

// NewServer creates a specflow MCP server with all tools registered.
func NewServer(s *store.Store, cfg config.Config, version string) *Server {
	srv := &Server{
		store:   s,
		cfg:     cfg,
		builder: sfcontext.New(s, cfg),
	}

	srv.mcpSrv = mcp.NewServer(
		&mcp.Implementation{Name: "specflow", Version: version},
		nil,
	)

	srv.registerReadTools()
	srv.registerWriteTools()

	return srv
}

// Run starts the MCP server on stdio.
func (s *Server) Run(ctx context.Context) error {
	return s.mcpSrv.Run(ctx, &mcp.StdioTransport{})
}

// textResult creates a CallToolResult with text content.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// errResult creates a CallToolResult with an error.
func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

// errResultf creates a formatted error result.
func errResultf(format string, args ...any) *mcp.CallToolResult {
	return errResult(fmt.Sprintf(format, args...))
}

// countStatuses tallies stories by status.
func countStatuses(stories []*models.Story) map[string]int {
	counts := make(map[string]int)
	for _, st := range stories {
		counts[st.Status]++
	}
	return counts
}

// containsString checks if a slice contains a string.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
