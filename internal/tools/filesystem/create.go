package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iSundram/ORTEDS/internal/config"
	"github.com/iSundram/ORTEDS/internal/diagnostics"
)

// CreateResult is returned by CreateFile.
type CreateResult struct {
	Content     string
	Diagnostics []diagnostics.Diagnostic
	Created     bool
	IsError     bool
}

// CreateFile creates a new file at path with the provided content.  If the
// file already exists it returns an error.  Diagnostics are run against the
// new content and included in the response.
func CreateFile(path, content string) CreateResult {
	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return CreateResult{
			Content: fmt.Sprintf("error: cannot create directories for %q: %s", path, err),
			IsError: true,
		}
	}

	// Fail if the file already exists.
	if _, err := os.Stat(path); err == nil {
		return CreateResult{
			Content: fmt.Sprintf("error: file %q already exists", path),
			IsError: true,
		}
	}

	cfg := config.Load()

	var diags []diagnostics.Diagnostic
	if cfg.Diagnostics.Enabled {
		diags = diagnostics.Analyze(path, content)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return CreateResult{
			Content: fmt.Sprintf("error: cannot create file %q: %s", path, err),
			IsError: true,
		}
	}

	header := formatDiagnosticsHeader(diags)
	msg := fmt.Sprintf("%sFile created: %q\n", header, path)

	return CreateResult{
		Content:     msg,
		Diagnostics: diags,
		Created:     true,
	}
}
