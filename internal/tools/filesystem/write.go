package filesystem

import (
	"fmt"
	"os"

	"github.com/iSundram/ORTEDS/internal/config"
	"github.com/iSundram/ORTEDS/internal/diagnostics"
)

// WriteResult is returned by WriteFile.
type WriteResult struct {
	Content     string
	Diagnostics []diagnostics.Diagnostic
	Applied     bool
	IsError     bool
}

// WriteFile writes content to path. Unlike EditFile the write is non-blocking:
// it always persists the file but still surfaces diagnostics in the response so
// the AI can be aware of any issues it has introduced.
func WriteFile(path, content string, opts EditOptions) WriteResult {
	cfg := config.Load()

	var diags []diagnostics.Diagnostic
	if cfg.Diagnostics.Enabled && !opts.ForceWrite {
		diags = diagnostics.Analyze(path, content)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return WriteResult{
			Content: fmt.Sprintf("error: cannot write file %q: %s", path, err),
			IsError: true,
		}
	}

	header := formatDiagnosticsHeader(diags)
	msg := fmt.Sprintf("%sFile written: %q\n", header, path)

	return WriteResult{
		Content:     msg,
		Diagnostics: diags,
		Applied:     true,
	}
}
