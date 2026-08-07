// Command youtube-mcp is an MCP stdio server exposing YouTube
// transcript/metadata/download tools.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/johncegom/go-youtube-mcp-cli/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is overridable at build time via:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/youtube-mcp
var version = "dev"

func main() {
	server := mcpserver.NewServer(version)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err)
		os.Exit(1)
	}
}
