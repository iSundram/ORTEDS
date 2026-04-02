package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iSundram/ORTEDS/internal/tools/filesystem"
)

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTestFile: %s", err)
	}
	return path
}

// ─── ReadFile ────────────────────────────────────────────────────────────────

func TestReadFile_ShowsDiagnosticsHeader(t *testing.T) {
	path := writeTestFile(t, "test.go", `func main() {
    println("test"
}
`)
	result := filesystem.ReadFile(path)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	// Should contain the DIAGNOSTICS header.
	if !containsStr(result.Content, "[DIAGNOSTICS") {
		t.Errorf("expected DIAGNOSTICS header in output, got:\n%s", result.Content)
	}
}

func TestReadFile_ValidFile_NoErrors(t *testing.T) {
	path := writeTestFile(t, "test.go", `package main

func main() {}
`)
	result := filesystem.ReadFile(path)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if containsStr(result.Content, "error(s) found") {
		t.Errorf("did not expect errors in valid Go file, got:\n%s", result.Content)
	}
}

func TestReadFile_MissingFile(t *testing.T) {
	result := filesystem.ReadFile("/nonexistent/path/test.go")
	if !result.IsError {
		t.Error("expected IsError=true for missing file")
	}
}

// ─── EditFile ────────────────────────────────────────────────────────────────

func TestEditFile_FixError_Applied(t *testing.T) {
	path := writeTestFile(t, "test.go", `package main

func main() {
    println("test"
}
`)
	result := filesystem.EditFile(path, `println("test"`, `println("test")`, filesystem.EditOptions{})
	if result.IsError {
		t.Fatalf("unexpected hard error: %s", result.Content)
	}
	if !result.Applied {
		t.Errorf("expected edit to be applied, got: %s", result.Content)
	}
	if result.Delta.FixedCount == 0 {
		t.Error("expected at least one error to be fixed")
	}
}

func TestEditFile_BlocksIntroducedError(t *testing.T) {
	path := writeTestFile(t, "test.go", `package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`)
	// Remove the package declaration – this should introduce an error.
	result := filesystem.EditFile(path, "package main\n\n", "", filesystem.EditOptions{})
	if result.IsError {
		t.Fatalf("unexpected hard error: %s", result.Content)
	}
	if result.Applied {
		t.Errorf("expected edit to be blocked, but it was applied")
	}
	if result.Delta.IntroducedCount == 0 {
		t.Error("expected at least one introduced error")
	}
}

func TestEditFile_ForceWrite_IgnoresValidation(t *testing.T) {
	path := writeTestFile(t, "test.go", `package main

func main() {}
`)
	// This edit introduces an error, but ForceWrite should bypass the check.
	result := filesystem.EditFile(path, "package main\n\n", "", filesystem.EditOptions{
		ForceWrite: true,
	})
	if result.IsError {
		t.Fatalf("unexpected hard error: %s", result.Content)
	}
	if !result.Applied {
		t.Errorf("expected force write to apply the edit")
	}
}

func TestEditFile_OldStrNotFound(t *testing.T) {
	path := writeTestFile(t, "test.go", `package main
func main() {}
`)
	result := filesystem.EditFile(path, "this string does not exist", "replacement", filesystem.EditOptions{})
	if !result.IsError {
		t.Error("expected error when old_str not found")
	}
}

// ─── WriteFile ───────────────────────────────────────────────────────────────

func TestWriteFile_AlwaysWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.go")

	result := filesystem.WriteFile(path, `package main
func main() {}
`, filesystem.EditOptions{})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !result.Applied {
		t.Error("expected Applied=true")
	}

	data, _ := os.ReadFile(path)
	if !containsStr(string(data), "package main") {
		t.Error("expected file to contain written content")
	}
}

// ─── CreateFile ──────────────────────────────────────────────────────────────

func TestCreateFile_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "created.go")

	result := filesystem.CreateFile(path, `package main
func main() {}
`)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !result.Created {
		t.Error("expected Created=true")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist on disk")
	}
}

func TestCreateFile_ExistingFileReturnsError(t *testing.T) {
	path := writeTestFile(t, "existing.go", "package main\n")
	result := filesystem.CreateFile(path, "package main\nfunc main(){}\n")
	if !result.IsError {
		t.Error("expected error when file already exists")
	}
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		findInStr(s, substr))
}

func findInStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
