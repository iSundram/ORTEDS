// orteds is the OweCode Realtime Tree-sitter Error Detection System CLI.
//
// Usage:
//
//	orteds analyze <file>                           - Analyze a file for errors
//	orteds read    <file>                           - Read file with diagnostics header
//	orteds edit    <file> <old_str> <new_str>       - Edit file with pre-flight validation
//	orteds write   <file> <content>                 - Write file (non-blocking)
//	orteds create  <file> <content>                 - Create new file
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/iSundram/ORTEDS/internal/diagnostics"
	"github.com/iSundram/ORTEDS/internal/tools/filesystem"
)

func main() {
	var (
		forceWrite          = flag.Bool("force-write", false, "bypass pre-flight validation when editing files")
		noDiagnostics       = flag.Bool("no-diagnostics", false, "disable all diagnostics")
		verboseDiagnostics  = flag.Bool("verbose-diagnostics", false, "print raw JSON diagnostics to stderr")
		outputJSON          = flag.Bool("json", false, "output results as JSON")
	)
	flag.Usage = usage
	flag.Parse()

	if *noDiagnostics {
		os.Setenv("ORTEDS_NO_DIAGNOSTICS", "1")
	}
	_ = verboseDiagnostics // used below via flag reference

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "analyze":
		runAnalyze(cmdArgs, *outputJSON)
	case "read":
		runRead(cmdArgs)
	case "edit":
		runEdit(cmdArgs, *forceWrite)
	case "write":
		runWrite(cmdArgs, *forceWrite)
	case "create":
		runCreate(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

// ─── Commands ────────────────────────────────────────────────────────────────

func runAnalyze(args []string, asJSON bool) {
	if len(args) < 1 {
		fatal("analyze: missing file argument")
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		fatal("analyze: cannot read %q: %s", path, err)
	}

	diags := diagnostics.Analyze(path, string(data))

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(diags); err != nil {
			fatal("analyze: json encode: %s", err)
		}
		return
	}

	if len(diags) == 0 {
		fmt.Printf("[DIAGNOSTICS: No errors found ✓]\n")
		return
	}

	fmt.Printf("[DIAGNOSTICS: %d error(s) found]\n", len(diags))
	for _, d := range diags {
		fmt.Printf("  %s Line %d Col %d: [%s] %s  (%s)\n",
			severityIcon(d.Severity), d.Line, d.Column, d.Code, d.Message, d.Source)
	}
	os.Exit(1) // non-zero exit when errors found
}

func runRead(args []string) {
	if len(args) < 1 {
		fatal("read: missing file argument")
	}
	result := filesystem.ReadFile(args[0])
	fmt.Print(result.Content)
	if result.IsError {
		os.Exit(1)
	}
}

func runEdit(args []string, forceWrite bool) {
	if len(args) < 3 {
		fatal("edit: usage: orteds edit <file> <old_str> <new_str>")
	}
	result := filesystem.EditFile(args[0], args[1], args[2], filesystem.EditOptions{
		ForceWrite: forceWrite,
	})
	fmt.Print(result.Content)
	if result.IsError || !result.Applied {
		os.Exit(1)
	}
}

func runWrite(args []string, forceWrite bool) {
	if len(args) < 2 {
		fatal("write: usage: orteds write <file> <content>")
	}
	result := filesystem.WriteFile(args[0], args[1], filesystem.EditOptions{
		ForceWrite: forceWrite,
	})
	fmt.Print(result.Content)
	if result.IsError {
		os.Exit(1)
	}
}

func runCreate(args []string) {
	if len(args) < 2 {
		fatal("create: usage: orteds create <file> <content>")
	}
	result := filesystem.CreateFile(args[0], args[1])
	fmt.Print(result.Content)
	if result.IsError {
		os.Exit(1)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func severityIcon(s string) string {
	if s == "error" {
		return "❌"
	}
	return "⚠️ "
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "orteds: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `orteds - OweCode Realtime Tree-sitter Error Detection System

Usage:
  orteds [flags] <command> [args]

Commands:
  analyze <file>                     Analyze file and print diagnostics
  read    <file>                     Read file with diagnostics header
  edit    <file> <old_str> <new_str> Edit file with pre-flight validation
  write   <file> <content>           Write file (non-blocking diagnostics)
  create  <file> <content>           Create new file with diagnostics

Flags:
  --force-write         Bypass pre-flight validation (always apply edits)
  --no-diagnostics      Disable all diagnostics
  --json                Output diagnostics as JSON (analyze command only)

Supported languages:
  Go (.go), Python (.py), JavaScript (.js/.jsx), TypeScript (.ts/.tsx),
  Rust (.rs), Java (.java), C (.c/.h), C++ (.cpp/.hpp/.cc),
  C# (.cs), Ruby (.rb), PHP (.php), Swift (.swift), Kotlin (.kt),
  Lua (.lua), JSON (.json), YAML (.yaml/.yml), TOML (.toml)

Configuration (~/.owecode/config.yaml):
  diagnostics:
    enabled: true
    show_in_read: true
    block_on_error: true
    block_on_warning: false
    max_file_size: 1048576   # 1 MB
    cache_duration: 30       # seconds
`)
}
