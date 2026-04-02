package filesystem

import (
	"fmt"
	"os"
	"strings"

	"github.com/iSundram/ORTEDS/internal/config"
	"github.com/iSundram/ORTEDS/internal/diagnostics"
)

// EditOptions controls optional behaviour of EditFile.
type EditOptions struct {
	// ForceWrite bypasses the pre-flight validation and always applies the edit.
	ForceWrite bool
}

// EditResult is returned by EditFile.
type EditResult struct {
	// Content is the formatted response message.
	Content string
	// Delta contains the before/after diagnostic comparison.
	Delta diagnostics.DiagnosticDelta
	// Applied is true when the edit was written to disk.
	Applied bool
	// IsError is true when a hard error occurred (e.g. file not readable).
	IsError bool
}

// EditFile reads path, replaces the first occurrence of oldStr with newStr,
// validates the result, and writes the file only when the change does not
// introduce new errors (unless ForceWrite is set).
func EditFile(path, oldStr, newStr string, opts EditOptions) EditResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return EditResult{
			Content: fmt.Sprintf("error: cannot read file %q: %s", path, err),
			IsError: true,
		}
	}

	current := string(data)

	if !strings.Contains(current, oldStr) {
		return EditResult{
			Content: fmt.Sprintf("error: old_str not found in %q", path),
			IsError: true,
		}
	}

	newContent := strings.Replace(current, oldStr, newStr, 1)
	cfg := config.Load()

	if !cfg.Diagnostics.Enabled || opts.ForceWrite {
		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return EditResult{
				Content: fmt.Sprintf("error: cannot write file %q: %s", path, err),
				IsError: true,
			}
		}
		return EditResult{
			Content: fmt.Sprintf("Change applied to %q (diagnostics disabled / force-write).", path),
			Applied: true,
		}
	}

	// Pre-flight validation.
	beforeDiags := diagnostics.Analyze(path, current)
	afterDiags := diagnostics.Analyze(path, newContent)
	delta := diagnostics.Compare(beforeDiags, afterDiags)

	if !delta.IsSafe && cfg.Diagnostics.BlockOnError {
		return EditResult{
			Content: formatValidationFailed(delta),
			Delta:   delta,
			Applied: false,
		}
	}

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return EditResult{
			Content: fmt.Sprintf("error: cannot write file %q: %s", path, err),
			IsError: true,
		}
	}

	return EditResult{
		Content: formatValidationPassed(delta),
		Delta:   delta,
		Applied: true,
	}
}
