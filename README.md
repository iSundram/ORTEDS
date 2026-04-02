# ORTEDS — OweCode Realtime Tree-sitter Error Detection System

> **Give AI eyes to see code errors — zero setup required.**

ORTEDS embeds [tree-sitter](https://tree-sitter.github.io) parsers directly into a single Go binary.
When an AI agent reads or edits a file, it automatically receives a diagnostics header that shows every syntax error **before** a broken change can reach disk.

---

## Features

| | |
|---|---|
| ✅ **Zero external dependencies** | No LSP, no language runtimes, no extra tools |
| ✅ **One binary, 18 languages** | All parsers statically linked via CGO |
| ✅ **Fast** | Syntax parsing < 50 ms; 30 s LRU cache |
| ✅ **Pre-flight edit validation** | Blocks edits that introduce new errors |
| ✅ **Deterministic** | Same results on every machine |

---

## Supported Languages

| Tier | Language | Extensions |
|------|----------|------------|
| 1 | Go | `.go` |
| 1 | Python | `.py` |
| 1 | JavaScript / JSX | `.js` `.jsx` |
| 1 | TypeScript / TSX | `.ts` `.tsx` |
| 1 | Rust | `.rs` |
| 1 | Java | `.java` |
| 2 | C | `.c` `.h` |
| 2 | C++ | `.cpp` `.hpp` `.cc` |
| 2 | C# | `.cs` |
| 2 | Ruby | `.rb` |
| 2 | PHP | `.php` |
| 3 | Swift | `.swift` |
| 3 | Kotlin | `.kt` `.kts` |
| 3 | Lua | `.lua` |
| Config | JSON | `.json` |
| Config | YAML | `.yaml` `.yml` |
| Config | TOML | `.toml` |

---

## Installation

**Requirements:** Go 1.21+, a C compiler (gcc or clang).

```bash
git clone https://github.com/iSundram/ORTEDS
cd ORTEDS
CGO_ENABLED=1 go build -o orteds ./cmd/orteds
```

---

## CLI Usage

```
orteds [flags] <command> [args]

Commands:
  analyze <file>                      Analyze file and print diagnostics
  read    <file>                      Read file with diagnostics header
  edit    <file> <old_str> <new_str>  Edit file with pre-flight validation
  write   <file> <content>            Write file (non-blocking diagnostics)
  create  <file> <content>            Create new file with diagnostics

Flags:
  --force-write         Bypass pre-flight validation
  --no-diagnostics      Disable all diagnostics
  --json                Output diagnostics as JSON (analyze only)
```

### Examples

**Analyze a file:**
```bash
$ orteds analyze main.go
[DIAGNOSTICS: 2 error(s) found]
  ❌ Line 2 Col 18: [missing-token] missing )  (tree-sitter-go)
  ❌ Line 1 Col 0: [missing-package] Go files must start with a 'package' declaration  (tree-sitter-go)
```

**Read with diagnostics header:**
```
$ orteds read main.go
═══════════════════════════════════
[DIAGNOSTICS: 1 error(s) found]
ERROR Line 6: missing-token - missing )
═══════════════════════════════════
   1. package main
   2. 
   3. import "fmt"
   4. 
   5. func main() {
   6.     fmt.Println("Hello"  // <-- missing closing paren
   7. }
```

**Edit with pre-flight validation (success):**
```
$ orteds edit main.go 'fmt.Println("Hello"' 'fmt.Println("Hello")'
═══════════════════════════════════
[VALIDATION PASSED ✓]

Impact:
  Fixed:      1 error(s)
  Introduced: 0 new error(s)
  Remaining:  0 error(s) in file

Change has been applied to disk.
═══════════════════════════════════
```

**Edit blocked (would introduce errors):**
```
$ orteds edit main.go 'package main' ''
═══════════════════════════════════
[VALIDATION FAILED ✗]

Impact:
  Fixed:      0 error(s)
  Introduced: 1 NEW error(s)
    Line 1: missing-package - Go files must start with a 'package' declaration

Change was NOT applied. File unchanged.
Try again with a different edit.
═══════════════════════════════════
```

**JSON output:**
```bash
$ orteds --json analyze main.go
[
  {
    "line": 6,
    "column": 4,
    "severity": "error",
    "code": "missing-token",
    "message": "missing )",
    "source": "tree-sitter-go"
  }
]
```

---

## Configuration

Create `~/.owecode/config.yaml`:

```yaml
diagnostics:
  enabled: true          # Enable/disable diagnostics
  show_in_read: true     # Prepend diagnostics header to read output
  block_on_error: true   # Block edit_file when new errors are introduced
  block_on_warning: false
  max_file_size: 1048576 # Skip files larger than 1 MB
  cache_duration: 30     # Cache parse results for 30 seconds
```

---

## Library Usage

ORTEDS can be embedded directly in other Go programs:

```go
import (
    "github.com/iSundram/ORTEDS/internal/diagnostics"
    "github.com/iSundram/ORTEDS/internal/tools/filesystem"
)

// Analyze a file
diags := diagnostics.Analyze("main.go", content)

// Compare before/after (for edit validation)
delta := diagnostics.Compare(beforeDiags, afterDiags)
if !delta.IsSafe {
    // new errors would be introduced
}

// High-level file tools
result := filesystem.ReadFile("main.go")
result := filesystem.EditFile("main.go", oldStr, newStr, filesystem.EditOptions{})
result := filesystem.WriteFile("main.go", newContent, filesystem.EditOptions{})
result := filesystem.CreateFile("newfile.go", content)
```

---

## Architecture

```
┌─────────────────────────────────────────┐
│         ORTEDS Binary (Single File)      │
├─────────────────────────────────────────┤
│  1. CLI / Tool Layer                    │
│     ↓ calls                              │
│  2. Diagnostic Engine                    │
│     ├─ Language Registry (extension→ID) │
│     ├─ Parse Cache (30 s TTL)           │
│     ├─ DiagnosticDelta (Compare)        │
│     ↓ uses                               │
│  3. Parser Registry (18 languages)      │
│     ├→ Tree-sitter Go (C lib via CGO)  │
│     ├→ Tree-sitter Python              │
│     ├→ Tree-sitter JS/TS              │
│     ├→ ... (14 more)                   │
│     ↓ produces                           │
│  4. Syntax Tree → ERROR node finder     │
│     ↓ outputs                            │
│  5. []Diagnostic (JSON-serialisable)    │
└─────────────────────────────────────────┘
```

### Data Types

```go
type Diagnostic struct {
    FilePath string
    Line     int    // 1-indexed
    Column   int    // 0-indexed
    Severity string // "error" | "warning"
    Code     string // "syntax-error" | "missing-token" | ...
    Message  string // human-readable
    Source   string // "tree-sitter-go" | "json-parser" | ...
}

type DiagnosticDelta struct {
    Fixed           []Diagnostic
    Introduced      []Diagnostic
    Unchanged       []Diagnostic
    FixedCount      int
    IntroducedCount int
    IsSafe          bool // true when IntroducedCount == 0
}
```

---

## AI System Prompt

Add this to your AI agent's system prompt:

```
DIAGNOSTIC SYSTEM:
When you call read_file you will see a [DIAGNOSTICS] header showing syntax errors.
When you call edit_file your changes are validated before being written:
  ✅ If your edit fixes errors without introducing new ones → Applied
  ❌ If your edit introduces new errors → BLOCKED (try again)

Guidelines:
1. Always inspect the [DIAGNOSTICS] header when reading files.
2. Fix errors you see before adding new features.
3. If an edit is blocked, analyse the introduced error and retry.
4. Do not remove code that would cause new errors elsewhere.
```

---

## Running Tests

```bash
CGO_ENABLED=1 go test ./...
```

---

## License

MIT
