// Package filesystem provides file-system tools with integrated diagnostics.
package filesystem

import (
	"fmt"
	"strings"

	"github.com/iSundram/ORTEDS/internal/diagnostics"
)

// formatDiagnosticsHeader builds the [DIAGNOSTICS] banner shown above file
// content in read/edit responses.
func formatDiagnosticsHeader(diags []diagnostics.Diagnostic) string {
	if len(diags) == 0 {
		return "═══════════════════════════════════\n[DIAGNOSTICS: No errors found ✓]\n═══════════════════════════════════\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "═══════════════════════════════════\n")
	fmt.Fprintf(&b, "[DIAGNOSTICS: %d error(s) found]\n", len(diags))
	for _, d := range diags {
		icon := "ERROR"
		if d.Severity == "warning" {
			icon = "WARN "
		}
		fmt.Fprintf(&b, "%s Line %d: %s - %s\n", icon, d.Line, d.Code, d.Message)
	}
	fmt.Fprintf(&b, "═══════════════════════════════════\n")
	return b.String()
}

// formatValidationPassed formats a successful edit validation result.
func formatValidationPassed(delta diagnostics.DiagnosticDelta) string {
	var b strings.Builder
	fmt.Fprintln(&b, "═══════════════════════════════════")
	fmt.Fprintln(&b, "[VALIDATION PASSED ✓]")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Impact:")
	fmt.Fprintf(&b, "  Fixed:      %d error(s)\n", delta.FixedCount)
	fmt.Fprintf(&b, "  Introduced: %d new error(s)\n", delta.IntroducedCount)
	fmt.Fprintf(&b, "  Remaining:  %d error(s) in file\n", len(delta.Unchanged))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Change has been applied to disk.")
	fmt.Fprintln(&b, "═══════════════════════════════════")
	return b.String()
}

// formatValidationFailed formats a blocked edit result.
func formatValidationFailed(delta diagnostics.DiagnosticDelta) string {
	var b strings.Builder
	fmt.Fprintln(&b, "═══════════════════════════════════")
	fmt.Fprintln(&b, "[VALIDATION FAILED ✗]")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Impact:")
	fmt.Fprintf(&b, "  Fixed:      %d error(s)\n", delta.FixedCount)
	fmt.Fprintf(&b, "  Introduced: %d NEW error(s)\n", delta.IntroducedCount)
	for _, d := range delta.Introduced {
		fmt.Fprintf(&b, "    Line %d: %s - %s\n", d.Line, d.Code, d.Message)
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Change was NOT applied. File unchanged.")
	fmt.Fprintln(&b, "Try again with a different edit.")
	fmt.Fprintln(&b, "═══════════════════════════════════")
	return b.String()
}

// addLineNumbers prepends 1-indexed line numbers to content.
func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%4d. %s\n", i+1, line)
	}
	return b.String()
}
