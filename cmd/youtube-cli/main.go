// Command youtube-cli prints YouTube transcripts/metadata/search results and
// downloads video/audio from the command line.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/johncegom/go-youtube-mcp-cli/internal/cli"
)

// version is overridable at build time via:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/youtube-cli
var version = "dev"

func main() {
	root := cli.NewRootCommand(version)
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
