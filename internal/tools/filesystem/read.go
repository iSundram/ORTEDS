package filesystem

import (
	"fmt"
	"os"

	"github.com/iSundram/ORTEDS/internal/config"
	"github.com/iSundram/ORTEDS/internal/diagnostics"
)

// ReadResult is returned by ReadFile.
type ReadResult struct {
	// Content is the (possibly annotated) file content.
	Content string
	// Diagnostics contains the parsed diagnostics for the file.
	Diagnostics []diagnostics.Diagnostic
	// IsError is true when the read itself failed.
	IsError bool
}

// ReadFile reads the file at path and, when diagnostics are enabled, prepends
// a diagnostics header to the returned content.
func ReadFile(path string) ReadResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReadResult{
			Content: fmt.Sprintf("error: cannot read file %q: %s", path, err),
			IsError: true,
		}
	}

	content := string(data)
	cfg := config.Load()

	if !cfg.Diagnostics.Enabled || !cfg.Diagnostics.ShowInRead {
		return ReadResult{Content: content}
	}

	diags := diagnostics.Analyze(path, content)

	var header string
	if len(diags) > 0 || cfg.Diagnostics.ShowInRead {
		header = formatDiagnosticsHeader(diags)
	}

	return ReadResult{
		Content:     header + addLineNumbers(content),
		Diagnostics: diags,
	}
}
