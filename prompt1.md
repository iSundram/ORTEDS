# OweCode Embedded Error Detection System (OEDS)

## 1. MISSION: Give AI Eyes to See Code Errors - Zero Setup Required

**Problem:** AI agents write code blindly. They can't see syntax errors, type mismatches, or undefined variables until the code is written and the user runs a compiler/linter. This creates a frustrating loop of:
1. AI writes code
2. User runs it
3. Errors appear
4. User reports errors back to AI
5. AI tries again (repeat)

**Current Solution (Other Tools):** "Install LSP servers, configure them, set up your environment..."  
❌ **Problem:** Users don't want to install 15 different tools. They want ONE binary that just works.

**Our Solution:** Embed all parsers directly into the OweCode binary using Tree-sitter + CGO.

### 1.1 Design Philosophy

✅ **Zero external dependencies** - No LSP, no compilers, no language runtimes required  
✅ **One binary, 15 languages** - All parsers statically linked into OweCode  
✅ **Works everywhere** - Linux, macOS, Windows (any x86_64/arm64)  
✅ **Fast** - Syntax parsing < 50ms, semantic checks < 200ms  
✅ **Deterministic** - Same errors on every machine, regardless of environment

### 1.2 Core Workflow

**When AI reads a file:**
```
AI calls: read_file("src/main.go")

Response:
---
[DIAGNOSTICS: 2 errors found]
Line 15: syntax error - undefined: fmt.Printl (did you mean fmt.Println?)
Line 23: syntax error - expected '}', found 'EOF'
---
<file content>
```

**When AI proposes a change:**
```
AI calls: edit_file(path="src/main.go", old_str="fmt.Printl", new_str="fmt.Println")

Pre-flight validation (in-memory):
✓ Parse current file → 2 errors
✓ Apply change in memory → Parse again → 1 error
✓ Compare: Fixed 1, Introduced 0

Response:
---
[VALIDATION RESULT]
✅ SAFE TO APPLY
Errors fixed: 1 (undefined: fmt.Printl)
Errors introduced: 0
---
```

**If AI introduces errors:**
```
AI calls: edit_file(path="src/main.go", old_str="func main() {", new_str="func main()")

Response:
---
[VALIDATION RESULT]
❌ BLOCKED - Would introduce errors
Errors introduced: 1 (missing function body)

Change was NOT applied. Try again.
---
```

---

## 2. ARCHITECTURE: Embedded Parser System

### 2.1 Core Components

```
┌─────────────────────────────────────────┐
│         OweCode Binary (Single File)     │
├─────────────────────────────────────────┤
│  1. Tool Layer (read/edit/write)        │
│     ↓ calls                              │
│  2. Diagnostic Engine                    │
│     ↓ uses                               │
│  3. Parser Registry (15 languages)       │
│     ├→ Tree-sitter Go (C lib)           │
│     ├→ Tree-sitter Python (C lib)       │
│     ├→ Tree-sitter JavaScript (C lib)   │
│     ├→ ... (12 more)                    │
│     ↓ produces                           │
│  4. Syntax Tree → Error Finder          │
│     ↓ outputs                            │
│  5. Diagnostic Results (JSON)            │
└─────────────────────────────────────────┘
```

### 2.2 How It Works

**Step 1: File Extension Detection**
```go
".go"   → Use tree-sitter-go
".py"   → Use tree-sitter-python
".js"   → Use tree-sitter-javascript
".ts"   → Use tree-sitter-typescript
".rs"   → Use tree-sitter-rust
// ... etc
```

**Step 2: Parsing (via CGO)**
```go
// Embedded C library via CGO
/*
#cgo CFLAGS: -Ivendor/tree-sitter-go/src
#include "parser.c"
*/
import "C"

func ParseGo(content string) *Tree {
    parser := C.ts_parser_new()
    C.ts_parser_set_language(parser, C.tree_sitter_go())
    tree := C.ts_parser_parse_string(parser, nil, content, len(content))
    return tree
}
```

**Step 3: Error Detection**
```go
func FindErrors(tree *Tree) []Diagnostic {
    errors := []Diagnostic{}
    
    // Method 1: Find ERROR nodes
    WalkTree(tree, func(node *Node) {
        if node.Type == "ERROR" || node.IsMissing {
            errors = append(errors, Diagnostic{
                Line: node.StartLine,
                Message: "syntax error at " + node.Text,
            })
        }
    })
    
    // Method 2: Run language-specific queries
    if tree.Language == "go" {
        errors = append(errors, CheckGoSpecificRules(tree)...)
    }
    
    return errors
}
```

**Step 4: Delta Comparison**
```go
func ValidateEdit(path, oldContent, newContent string) ValidationResult {
    oldErrors := ParseAndGetErrors(path, oldContent)
    newErrors := ParseAndGetErrors(path, newContent)
    
    fixed := ErrorsIn(oldErrors) - ErrorsIn(newErrors)
    introduced := ErrorsIn(newErrors) - ErrorsIn(oldErrors)
    
    return ValidationResult{
        IsSafe: len(introduced) == 0,
        Fixed: fixed,
        Introduced: introduced,
    }
}
```

---

## 3. THE 15 SUPPORTED LANGUAGES

### Priority Tier 1 (Most Popular - Must Have)
1. **JavaScript** (`.js`, `.jsx`) - tree-sitter-javascript
2. **Python** (`.py`) - tree-sitter-python
3. **TypeScript** (`.ts`, `.tsx`) - tree-sitter-typescript
4. **Go** (`.go`) - tree-sitter-go
5. **Rust** (`.rs`) - tree-sitter-rust
6. **Java** (`.java`) - tree-sitter-java

### Priority Tier 2 (Common Languages)
7. **C** (`.c`, `.h`) - tree-sitter-c
8. **C++** (`.cpp`, `.hpp`, `.cc`) - tree-sitter-cpp
9. **C#** (`.cs`) - tree-sitter-c-sharp
10. **Ruby** (`.rb`) - tree-sitter-ruby
11. **PHP** (`.php`) - tree-sitter-php

### Priority Tier 3 (Modern/Growing)
12. **Swift** (`.swift`) - tree-sitter-swift
13. **Kotlin** (`.kt`) - tree-sitter-kotlin
14. **Zig** (`.zig`) - tree-sitter-zig
15. **Lua** (`.lua`) - tree-sitter-lua

### Bonus: Config Files (Syntax-only)
- **JSON** (`.json`) - tree-sitter-json
- **YAML** (`.yaml`, `.yml`) - tree-sitter-yaml
- **TOML** (`.toml`) - tree-sitter-toml

---

## 4. WHAT EACH LANGUAGE DETECTS

### 4.1 Universal (All Languages)
✅ Missing closing braces `}`  
✅ Missing closing brackets `]`  
✅ Missing closing parentheses `)`  
✅ Unclosed strings  
✅ Invalid escape sequences  
✅ Unexpected EOF (end of file)

### 4.2 Language-Specific Rules

**JavaScript/TypeScript:**
- ✅ Missing semicolons (optional, but detectable)
- ✅ Unclosed JSX tags `<div>` without `</div>`
- ✅ `await` outside `async` function
- ✅ Invalid object keys
- ✅ Missing `import` statement syntax

**Python:**
- ✅ Indentation errors (tabs vs spaces)
- ✅ Invalid decorator syntax `@`
- ✅ Missing colons after `if`, `for`, `def`
- ✅ Invalid f-string syntax
- ✅ Incomplete function definitions

**Go:**
- ✅ Missing `package` declaration
- ✅ Invalid import syntax
- ✅ Missing return statement type
- ✅ Invalid function signatures
- ✅ Unclosed blocks

**Rust:**
- ✅ Missing semicolons on statements
- ✅ Invalid lifetime syntax `'a`
- ✅ Unclosed macro calls `println!(...)`
- ✅ Invalid match arms
- ✅ Missing `use` statements

**Java:**
- ✅ Missing semicolons
- ✅ Class name must match filename
- ✅ Missing `public static void main`
- ✅ Invalid access modifiers
- ✅ Unclosed generic brackets `<T>`

**C/C++:**
- ✅ Missing semicolons
- ✅ Invalid pointer/reference syntax `*`, `&`
- ✅ Missing `#include` directives
- ✅ Unclosed class/struct definitions
- ✅ Invalid template syntax

**Python, Ruby, PHP:**
- ✅ Language-specific syntax (indentation, sigils, tags)

---

## 5. TOOL INTEGRATION: How It Works in Practice

### 5.1 Enhanced `read_file` Tool

**Current behavior:**
```go
read_file("app.py") → Returns file content only
```

**With diagnostics:**
```go
read_file("app.py")

Returns:
═══════════════════════════════════
[DIAGNOSTICS: 3 errors found]
ERROR Line 12: SyntaxError - invalid syntax
ERROR Line 18: IndentationError - expected indent
ERROR Line 25: SyntaxError - unclosed string

═══════════════════════════════════
1. import os
2. import sys
3. 
4. def main():
5.     print("hello"
...
```

### 5.2 Validated `edit_file` Tool

**Behavior change:**
```go
// AI proposes edit
edit_file(
    path="app.py",
    old_str='print("hello"',
    new_str='print("hello")'
)

// System validates in-memory BEFORE writing
Step 1: Parse current file → 3 errors
Step 2: Apply change to temp buffer
Step 3: Parse temp buffer → 2 errors
Step 4: Compare → Fixed 1, Introduced 0

Returns:
═══════════════════════════════════
[VALIDATION PASSED ✓]

Impact:
  Fixed: 1 error (unclosed string)
  Introduced: 0 new errors
  Remaining: 2 errors in file

Change has been applied to disk.
═══════════════════════════════════
```

**When blocked:**
```go
edit_file(
    path="app.py",
    old_str='def main():',
    new_str='def main()'
)

Returns:
═══════════════════════════════════
[VALIDATION FAILED ✗]

Impact:
  Fixed: 0 errors
  Introduced: 1 NEW error
    Line 12: SyntaxError - expected ':'

Change was NOT applied. File unchanged.
Try again with a different edit.
═══════════════════════════════════
```

### 5.3 Implementation Pseudocode

```go
// internal/tools/filesystem/write.go

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
    path := args["path"].(string)
    oldStr := args["old_str"].(string)
    newStr := args["new_str"].(string)
    
    // Read current file
    currentContent := readFile(path)
    
    // STEP 1: Analyze current errors
    currentDiags := diagnostics.Analyze(path, currentContent)
    
    // STEP 2: Simulate the change
    newContent := strings.Replace(currentContent, oldStr, newStr, 1)
    
    // STEP 3: Analyze new errors
    newDiags := diagnostics.Analyze(path, newContent)
    
    // STEP 4: Compare
    delta := diagnostics.Compare(currentDiags, newDiags)
    
    // STEP 5: Decision
    if delta.IntroducedCount > 0 {
        return tools.Result{
            IsError: true,
            Content: formatBlockedEdit(delta),
        }, nil
    }
    
    // STEP 6: Safe to write
    writeFile(path, newContent)
    
    return tools.Result{
        Content: formatSuccessfulEdit(delta),
    }, nil
}
```

---

## 6. DATA STRUCTURES

### 6.1 Diagnostic Format
```go
type Diagnostic struct {
    FilePath string
    Line     int
    Column   int
    Severity string  // "error" | "warning"
    Code     string  // "syntax-error" | "missing-brace" etc
    Message  string  // Human readable
    Source   string  // "tree-sitter-go" | "tree-sitter-python"
}
```

### 6.2 Comparison Delta
```go
type DiagnosticDelta struct {
    Fixed       []Diagnostic  // Errors gone after change
    Introduced  []Diagnostic  // New errors after change
    Unchanged   []Diagnostic  // Errors in both versions
    
    FixedCount      int
    IntroducedCount int
    IsSafe          bool  // true if IntroducedCount == 0
}

func Compare(before, after []Diagnostic) DiagnosticDelta {
    // Match by: Line + Code + Message
    beforeMap := toMap(before)
    afterMap := toMap(after)
    
    fixed := subtract(beforeMap, afterMap)
    introduced := subtract(afterMap, beforeMap)
    unchanged := intersect(beforeMap, afterMap)
    
    return DiagnosticDelta{
        Fixed: fixed,
        Introduced: introduced,
        Unchanged: unchanged,
        FixedCount: len(fixed),
        IntroducedCount: len(introduced),
        IsSafe: len(introduced) == 0,
    }
}
```

---

## 7. IMPLEMENTATION ROADMAP

### Phase 1: Core Infrastructure (Week 1-2)
**Goal:** Get Tree-sitter parsers embedded and working

**Tasks:**
1. ✅ Create `internal/diagnostics/` package
2. ✅ Add CGO build setup for Tree-sitter C libs
3. ✅ Embed parsers for Tier 1 languages (JS, Python, TS, Go, Rust, Java)
4. ✅ Write parser wrapper for each language
5. ✅ Implement ERROR node detection
6. ✅ Create diagnostic comparison logic
7. ✅ Write unit tests for each parser

**Deliverable:** `diagnostics.Analyze(path, content) → []Diagnostic`

---

### Phase 2: Tool Integration (Week 3)
**Goal:** Hook diagnostics into file tools

**Tasks:**
1. ✅ Modify `read_file` → Append diagnostics header
2. ✅ Modify `edit_file` → Add pre-flight validation
3. ✅ Modify `write_file` → Add validation (non-blocking)
4. ✅ Modify `create_file` → Validate new files
5. ✅ Add config option: `diagnostics.enabled` in config.yaml
6. ✅ Integration tests for all tools

**Deliverable:** Tools return diagnostic info to AI

---

### Phase 3: Language Expansion (Week 4-5)
**Goal:** Add remaining 9 languages

**Tasks:**
1. ✅ Add Tier 2 parsers (C, C++, C#, Ruby, PHP)
2. ✅ Add Tier 3 parsers (Swift, Kotlin, Zig, Lua)
3. ✅ Add config file parsers (JSON, YAML, TOML)
4. ✅ Write language-specific query rules (.scm files)
5. ✅ Test each language with known error files

---

### Phase 4: Polish & Optimization (Week 6)
**Goal:** Make it fast and user-friendly

**Tasks:**
1. ✅ Cache parse results (30s TTL)
2. ✅ Skip diagnostics for huge files (>1MB)
3. ✅ Improve error messages (add suggestions)
4. ✅ Add `--force-write` flag to bypass validation
5. ✅ Update AI system prompt
6. ✅ Write documentation

---

## 8. TECHNICAL IMPLEMENTATION DETAILS

### 8.1 CGO Build Setup

**Directory structure:**
```
internal/diagnostics/
├── diagnostics.go          # Main API
├── registry.go             # Language detection
├── compare.go              # Delta comparison
├── parsers/
│   ├── go.go              # Go parser wrapper
│   ├── python.go          # Python parser wrapper
│   ├── javascript.go      # JS parser wrapper
│   └── ...
└── vendor/
    ├── tree-sitter-go/
    │   └── src/
    │       ├── parser.c
    │       └── scanner.c
    ├── tree-sitter-python/
    │   └── src/
    │       └── parser.c
    └── ... (13 more)
```

**Build configuration:**
```go
// internal/diagnostics/parsers/go.go

package parsers

/*
#cgo CFLAGS: -Ivendor/tree-sitter-go/src -std=c11
#include <tree_sitter/parser.h>

extern const TSLanguage *tree_sitter_go(void);

#include "vendor/tree-sitter-go/src/parser.c"
*/
import "C"
import "unsafe"

func ParseGo(content string) (*Tree, error) {
    parser := C.ts_parser_new()
    defer C.ts_parser_delete(parser)
    
    lang := C.tree_sitter_go()
    C.ts_parser_set_language(parser, lang)
    
    cContent := C.CString(content)
    defer C.free(unsafe.Pointer(cContent))
    
    tree := C.ts_parser_parse_string(
        parser,
        nil,
        cContent,
        C.uint32_t(len(content)),
    )
    
    return wrapTree(tree), nil
}
```

### 8.2 Error Detection Algorithm

```go
// internal/diagnostics/analyze.go

func Analyze(path, content string) []Diagnostic {
    lang := detectLanguage(path)  // by file extension
    if lang == "" {
        return nil  // Unsupported language
    }
    
    // Step 1: Parse to syntax tree
    tree, err := parsers.Parse(lang, content)
    if err != nil {
        return []Diagnostic{{
            Line: 0,
            Message: "Failed to parse file",
            Severity: "error",
        }}
    }
    
    // Step 2: Find ERROR nodes
    errors := []Diagnostic{}
    walkTree(tree.Root(), func(node *Node) {
        if node.Type == "ERROR" {
            errors = append(errors, Diagnostic{
                Line: node.StartLine + 1,  // 1-indexed
                Column: node.StartColumn,
                Severity: "error",
                Code: "syntax-error",
                Message: fmt.Sprintf("syntax error: %s", node.Text()),
                Source: fmt.Sprintf("tree-sitter-%s", lang),
            })
        }
        
        if node.IsMissing {
            errors = append(errors, Diagnostic{
                Line: node.StartLine + 1,
                Column: node.StartColumn,
                Severity: "error",
                Code: "missing-token",
                Message: fmt.Sprintf("missing %s", node.Type),
                Source: fmt.Sprintf("tree-sitter-%s", lang),
            })
        }
    })
    
    // Step 3: Language-specific rules
    if lang == "go" {
        errors = append(errors, checkGoRules(tree)...)
    }
    if lang == "python" {
        errors = append(errors, checkPythonRules(tree)...)
    }
    // ... etc
    
    return errors
}
```

### 8.3 Language-Specific Rules

**Example: Go package declaration check**
```go
func checkGoRules(tree *Tree) []Diagnostic {
    errors := []Diagnostic{}
    
    // Rule 1: Must have package declaration
    root := tree.Root()
    if root.ChildCount() == 0 {
        return errors
    }
    
    firstChild := root.Child(0)
    if firstChild.Type != "package_clause" {
        errors = append(errors, Diagnostic{
            Line: 1,
            Severity: "error",
            Code: "missing-package",
            Message: "Go files must start with 'package' declaration",
        })
    }
    
    return errors
}
```

**Example: Python indentation check**
```go
func checkPythonRules(tree *Tree) []Diagnostic {
    errors := []Diagnostic{}
    
    // Rule 1: Check indentation consistency
    indentStack := []int{0}
    
    walkTree(tree.Root(), func(node *Node) {
        if node.Type == "block" {
            expectedIndent := indentStack[len(indentStack)-1] + 4
            actualIndent := node.StartColumn
            
            if actualIndent != expectedIndent {
                errors = append(errors, Diagnostic{
                    Line: node.StartLine + 1,
                    Severity: "error",
                    Code: "indentation-error",
                    Message: fmt.Sprintf("expected indent %d, got %d", expectedIndent, actualIndent),
                })
            }
            
            indentStack = append(indentStack, actualIndent)
        }
    })
    
    return errors
}
```

---

## 9. CONFIGURATION

### 9.1 User Config (`~/.owecode/config.yaml`)

```yaml
diagnostics:
  enabled: true                  # Enable/disable diagnostics
  show_in_read: true             # Show diagnostics in read_file
  block_on_error: true           # Block edit_file if new errors
  block_on_warning: false        # Don't block on warnings
  max_file_size: 1048576         # 1MB limit (skip larger files)
  cache_duration: 30             # Cache results for 30s
```

### 9.2 CLI Flags

```bash
# Disable diagnostics
owecode --no-diagnostics

# Force write even with errors
owecode --force-write

# Verbose diagnostic output
owecode --verbose-diagnostics
```

### 9.3 AI System Prompt Addition

```
DIAGNOSTIC SYSTEM:
When you read files with read_file, you'll see a [DIAGNOSTICS] header
showing any syntax errors in the code.

When you edit files with edit_file, your changes are validated:
- ✅ If your edit fixes errors without introducing new ones → Applied
- ❌ If your edit introduces new errors → BLOCKED (try again)

Guidelines:
1. Always check diagnostics when reading files
2. Fix errors you see
3. If an edit is blocked, analyze the error and retry
4. Don't remove code that would cause new errors elsewhere
```

---

## 10. EXAMPLE WORKFLOWS

### Example 1: Fix Syntax Error

**User:** "Fix errors in main.go"

**Step 1: AI reads file**
```
read_file("main.go")

Response:
═══════════════════════════════════
[DIAGNOSTICS: 1 error]
ERROR Line 6: syntax error - missing ')'

═══════════════════════════════════
1. package main
2. 
3. import "fmt"
4. 
5. func main() {
6.     fmt.Println("Hello"  // <-- missing closing paren
7. }
```

**Step 2: AI proposes fix**
```
edit_file(
    path="main.go",
    old_str='fmt.Println("Hello"',
    new_str='fmt.Println("Hello")'
)

Response:
═══════════════════════════════════
[VALIDATION PASSED ✓]

Impact:
  Fixed: 1 error (missing ')')
  Introduced: 0 errors

Change applied.
═══════════════════════════════════
```

---

### Example 2: Blocked Edit

**User:** "Remove fmt import"

**Step 1: AI attempts edit**
```
edit_file(
    path="main.go",
    old_str='import "fmt"\n',
    new_str=''
)

Response:
═══════════════════════════════════
[VALIDATION FAILED ✗]

Impact:
  Fixed: 0 errors
  Introduced: 1 NEW error
    Line 6: undefined identifier 'fmt'

Change blocked. File unchanged.
═══════════════════════════════════
```

**Step 2: AI adjusts strategy**
"Cannot remove import - it's being used. I'll need to either:
1. Remove both the import and the usage, or
2. Keep the import

Which would you prefer?"

---

### Example 3: Incremental Fixes

**User:** "Fix all errors in app.py"

```
read_file("app.py")

[DIAGNOSTICS: 3 errors]
ERROR Line 5: IndentationError
ERROR Line 12: SyntaxError - unclosed string
ERROR Line 18: SyntaxError - missing colon
```

**AI fixes one at a time:**

```
# Fix 1
edit_file(old_str="def func()\n  pass", new_str="def func():\n    pass")
✓ Fixed: 1 error (missing colon + indentation)

# Fix 2
edit_file(old_str='print("hello', new_str='print("hello")')
✓ Fixed: 1 error (unclosed string)

# Verify
read_file("app.py")
[DIAGNOSTICS: 0 errors] ✓ All clear!
```

---

## 11. TESTING STRATEGY

### 11.1 Unit Tests (Per Language)

**Create test files with known errors:**

```go
// internal/diagnostics/parsers/go_test.go

func TestGoParser_MissingBrace(t *testing.T) {
    code := `package main
func main() {
    println("test"
}` // Missing closing paren
    
    diags := diagnostics.Analyze("test.go", code)
    
    assert.Len(t, diags, 1)
    assert.Equal(t, 3, diags[0].Line)
    assert.Contains(t, diags[0].Message, "syntax error")
}

func TestGoParser_MissingPackage(t *testing.T) {
    code := `func main() {}`
    
    diags := diagnostics.Analyze("test.go", code)
    
    assert.Len(t, diags, 1)
    assert.Equal(t, "missing-package", diags[0].Code)
}
```

**Repeat for all 15 languages** - Each with 5-10 common error patterns.

---

### 11.2 Integration Tests (Tool Hooks)

```go
func TestReadFileShowsDiagnostics(t *testing.T) {
    writeTestFile("test.go", `package main
func main() {
    println("test"
}`)
    
    result := readFileTool.Execute(ctx, map[string]any{
        "path": "test.go",
    })
    
    assert.Contains(t, result.Content, "[DIAGNOSTICS")
    assert.Contains(t, result.Content, "syntax error")
}

func TestEditFileBlocksNewErrors(t *testing.T) {
    writeTestFile("test.go", `package main
import "fmt"
func main() {
    fmt.Println("ok")
}`)
    
    result := editFileTool.Execute(ctx, map[string]any{
        "path": "test.go",
        "old_str": "import \"fmt\"\n",
        "new_str": "",
    })
    
    assert.True(t, result.IsError)
    assert.Contains(t, result.Content, "BLOCKED")
}
```

---

### 11.3 End-to-End Tests

**Simulate full AI workflows:**

1. **Happy path:** Read file with errors → Fix → Validation passes
2. **Blocked:** Try bad edit → Blocked → Try again → Success
3. **Multiple errors:** Fix 3 errors incrementally
4. **All languages:** Test at least 2 error types per language

---

## 12. BUILD & DISTRIBUTION

### 12.1 CGO Build Requirements

**Dependencies:**
- C compiler (gcc/clang)
- Go 1.21+
- Tree-sitter C libraries (vendored)

**Build command:**
```bash
# Standard build
CGO_ENABLED=1 go build -o owecode ./cmd/owecode

# Static binary (Linux)
CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-linkmode external -extldflags -static" \
    -o owecode-linux ./cmd/owecode

# Cross-compilation setup needed for:
# - macOS: CGO_ENABLED=1 GOOS=darwin
# - Windows: CGO_ENABLED=1 GOOS=windows (requires mingw)
```

### 12.2 Binary Size Considerations

**Estimated sizes:**
- Base OweCode: ~15MB
- + Tree-sitter parsers (15 languages): ~5MB
- **Total: ~20MB** (compressed: ~7MB)

This is acceptable for a zero-dependency tool.

### 12.3 Vendoring Tree-sitter

```bash
# Script to vendor all parsers
./scripts/vendor-parsers.sh

# Downloads to:
internal/diagnostics/vendor/
├── tree-sitter-go/
├── tree-sitter-python/
├── tree-sitter-javascript/
└── ... (12 more)
```

---

## 13. SUMMARY & NEXT STEPS

### 13.1 What This Gives Users

**For AI:**
- 🔍 Real-time visibility into code errors
- 🛡️ Can't accidentally break working code
- 🎯 Knows when fixes actually work
- 🔄 Self-corrects when edits are blocked

**For Users:**
- ⚡ Fewer AI iterations needed
- ✅ Trust that AI won't introduce syntax errors
- 📦 Zero setup - works out of the box
- 🌍 Works on any machine (no LSP required)

### 13.2 Key Design Decisions

1. **All-in-one binary** - Tree-sitter parsers embedded via CGO
2. **15 languages** - Covers 95%+ of developer use cases
3. **Syntax-only** - Fast, deterministic, no semantic analysis needed
4. **Non-blocking** - Shows errors but doesn't prevent all operations
5. **Incremental** - Start with 6 languages, expand to 15

### 13.3 What's NOT Included

❌ **Semantic analysis** (types, imports, cross-file) - too complex  
❌ **Linting rules** (style, conventions) - use existing linters  
❌ **Auto-fixing** - AI decides fixes, not the system  
❌ **IDE features** (autocomplete, refactoring) - out of scope  

### 13.4 Implementation Estimate

**Phase 1 (Core + 6 languages):** 2 weeks  
**Phase 2 (Tool integration):** 1 week  
**Phase 3 (+9 languages):** 2 weeks  
**Phase 4 (Polish):** 1 week  

**Total: 6 weeks** for full 15-language support

### 13.5 Success Metrics

- ✅ Parse 99%+ of valid code without errors
- ✅ Catch 80%+ of syntax errors before write
- ✅ < 100ms parsing time for files under 2000 lines
- ✅ Binary size under 25MB
- ✅ Zero external dependencies

---

## 14. IMPLEMENTATION CHECKLIST

**Week 1-2: Core Infrastructure**
- [ ] Create `internal/diagnostics/` package structure
- [ ] Set up CGO build for Tree-sitter
- [ ] Vendor Tree-sitter C libraries for 6 languages
- [ ] Implement parser wrappers (Go, Python, JS, TS, Rust, Java)
- [ ] Implement ERROR node detection
- [ ] Implement diagnostic comparison logic
- [ ] Write unit tests for each parser

**Week 3: Tool Integration**
- [ ] Hook diagnostics into `read_file` tool
- [ ] Hook diagnostics into `edit_file` tool (pre-flight)
- [ ] Hook diagnostics into `write_file` tool
- [ ] Add config options to `config.yaml`
- [ ] Update AI system prompt
- [ ] Write integration tests

**Week 4-5: Language Expansion**
- [ ] Add 9 more parsers (C, C++, C#, Ruby, PHP, Swift, Kotlin, Zig, Lua)
- [ ] Add config file parsers (JSON, YAML, TOML)
- [ ] Write language-specific rules
- [ ] Test each language

**Week 6: Polish**
- [ ] Add caching (30s TTL)
- [ ] Add file size limits
- [ ] Improve error messages
- [ ] Add `--force-write` flag
- [ ] Write documentation
- [ ] Release!

---

**Document Version:** 3.0 (Zero-Dependency Edition)  
**Last Updated:** 2026-04-02  
**Status:** Ready for implementation  
**Estimated Binary Size:** 20MB (7MB compressed)  
**Target Languages:** 15 + 3 config formats
