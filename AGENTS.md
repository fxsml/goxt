# AI Coding Agent Guide

Guide for AI agents working on this codebase.

## Project Overview

`goxt` is a minimal task runner for Go projects with two modes:

1. **Mode A (CLI-only)**: Zero dependencies, tasks run via `goxt` CLI
2. **Mode B (Library)**: One dependency, adds `Deps()` and `go run` support

## Architecture

```
github.com/fxsml/goxt/
├── goxt.go              # Library: Run(), Deps()
├── cmd/goxt/main.go     # CLI binary
├── go.mod
├── README.md
└── AGENTS.md
```

### Library (`goxt.go`)

```go
package goxt

func Run(tasks ...func() error)           // Register and execute tasks
func Deps(tasks ...func() error) error    // Run dependencies once
```

### CLI (`cmd/goxt/main.go`)

```
main()
├── --init handling      # Create goxt.go (Mode A or B)
├── parseGoxtFile()      # Parse using go/ast, detect main()
├── printHelp()          # Show tasks with godoc descriptions
├── runGoxtFile()        # Has main() → go run goxt.go
└── runWithWrapper()     # No main() → generate temp wrapper
```

## Key Design Decisions

### Dual mode support
- Mode A: Zero deps, CLI generates temp wrapper with main()
- Mode B: Library import, user provides main() with `goxt.Run()`

### Stdlib only (for CLI)
Uses:
- `go/ast`, `go/parser`, `go/token` - Parse Go source
- `go/doc` - Extract documentation
- `os/exec` - Run `go run`

### Convention over configuration
- Exported `func() error` functions are tasks
- Godoc comments become descriptions
- CamelCase → kebab-case (`RunTests` → `run-tests`)

## Task Detection

A function is a task if:
1. Exported (uppercase first letter)
2. No receiver (not a method)
3. No parameters
4. Returns exactly `error`
5. Not named `Tasks` (reserved)

## Temp Wrapper Generation (Mode A)

When goxt.go has no `main()`, the CLI:
1. Reads goxt.go content
2. Appends a generated main() that calls the task
3. Writes to `.goxt_tmp.go`
4. Runs `go run .goxt_tmp.go`
5. Cleans up temp file

## Common Tasks

```bash
# Build everything
go build ./...

# Build and install CLI
go install ./cmd/goxt

# Test Mode A
cd /tmp && goxt --init && goxt build

# Test Mode B
cd /tmp && goxt --init --lib && go run goxt.go build
```

## Code Style

- Library is minimal (~100 lines)
- CLI handles all complexity
- No external dependencies
- Exit codes: 0 success, 1 error

## Extending

### Adding CLI flags
Add to `main()` with `hasFlag()` check.

### Adding library functions
Add to `goxt.go`, export for user access.

### Changing task detection
Modify `isTaskFunc()` in `cmd/goxt/main.go`.
