# AI Coding Agent Guide

Guide for AI agents working on this codebase.

## Project Overview

`goxt` is a minimal task runner for Go projects. It consists of:

1. **A convention**: `goxt.go` files with exported `func() error` functions as tasks
2. **An optional CLI**: `goxt` binary that parses `goxt.go` and provides enhanced help

## Architecture

```
main.go          # Single-file CLI (~250 lines)
├── main()       # Entry point, handles init/help/task dispatch
├── initGoxtFile() # Creates goxt.go from template
├── parseTasks() # Parses goxt.go using go/ast and go/doc
├── findGoxtFile() # Walks up directory tree to find goxt.go
└── printHelp()  # Displays tasks with descriptions
```

## Key Design Decisions

### Stdlib only
No external dependencies. Uses:
- `go/ast`, `go/parser`, `go/token` - Parse Go source
- `go/doc` - Extract documentation
- `os/exec` - Run `go run goxt.go`

### Convention over configuration
- Exported functions with `func() error` signature are tasks
- Godoc comments become descriptions
- CamelCase function names become kebab-case task names

### Backwards compatible
- `go run goxt.go <task>` always works (no goxt required)
- `goxt` just adds convenience (init, better help)

## The `goxt.go` Template

The template in `goxtTemplate` constant generates a working goxt.go with:
- Minimal boilerplate (~30 lines)
- Example tasks showing args and flags patterns
- `//go:build ignore` tag (excluded from normal builds, but `go run` ignores tags)

## Task Detection

A function is detected as a task if:
1. Exported (uppercase first letter)
2. No receiver (not a method)
3. No parameters
4. Returns exactly one value of type `error`
5. Not named `Tasks` (reserved)

## Common Tasks

```bash
# Build
go build .

# Test
go test .

# Run locally
go run . init      # Test init command
go run .           # Test help output
go run . <task>    # Test running a task
```

## Code Style

- Single file for simplicity
- No interfaces (concrete types only)
- Minimal error wrapping
- Exit codes: 0 success, 1 error

## Extending

### Adding goxt commands
Add handling in `main()` before `findGoxtFile()`:
```go
if len(os.Args) >= 2 && os.Args[1] == "newcmd" {
    // handle newcmd
    return
}
```

### Changing task detection
Modify `isTaskFunc()` to accept different signatures.

### Changing name conversion
Modify `camelToKebab()` for different naming conventions.
