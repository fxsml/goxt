# goxt

A minimal task runner for Go projects. Pure Go alternative to Makefile.

## Two Modes

| | Mode A: CLI-only | Mode B: Library |
|---|---|---|
| Dependencies | Zero | One (`github.com/fxsml/goxt`) |
| `goxt build` | ✓ | ✓ |
| `go run goxt.go` | ✗ | ✓ |
| Task dependencies | ✗ | ✓ (`goxt.Deps()`) |

Choose based on your needs:
- **Mode A**: Zero deps, use `goxt` CLI only
- **Mode B**: One dep, get `Deps()` and `go run` support

## Install CLI

```bash
go install github.com/fxsml/goxt/cmd/goxt@latest
```

## Quick Start

### Mode A: Zero Dependencies

```bash
goxt --init
goxt build
```

Creates a `goxt.go` with no external imports:

```go
//go:build ignore
package main

import "os/exec"

// Build compiles the project.
func Build() error {
    cmd := exec.Command("go", "build", "./...")
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    return cmd.Run()
}

// Test runs the tests.
func Test() error {
    cmd := exec.Command("go", "test", "./...")
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    return cmd.Run()
}
```

No `main()` needed - the CLI handles execution.

### Mode B: With Library

```bash
goxt --init --lib
goxt build
# or
go run goxt.go build
```

Creates a `goxt.go` with library import:

```go
//go:build ignore
package main

import "github.com/fxsml/goxt"

func main() {
    goxt.Run(Build, Test, Deploy)
}

// Build compiles the project.
func Build() error {
    cmd := exec.Command("go", "build", "./...")
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    return cmd.Run()
}

// Deploy runs Build and Test first, then deploys.
func Deploy() error {
    if err := goxt.Deps(Build, Test); err != nil {
        return err
    }
    fmt.Println("Deploying...")
    return nil
}
```

With `goxt.Deps()`, each task runs only once even with diamond dependencies.

## CLI Usage

```bash
goxt --init          # Create goxt.go (Mode A: zero deps)
goxt --init --lib    # Create goxt.go (Mode B: with library)
goxt                 # Show tasks with descriptions
goxt <task>          # Run a task
goxt <task> args...  # Run with arguments
```

## Adding Tasks

Just add exported functions that return `error`:

```go
// Deploy deploys to the specified environment.
func Deploy() error {
    env := "prod"
    if len(os.Args) >= 3 {
        env = os.Args[2]
    }
    fmt.Println("Deploying to", env)
    return nil
}
```

The CLI auto-discovers tasks by parsing the file.

## Task Dependencies (Mode B only)

```go
import "github.com/fxsml/goxt"

func Deploy() error {
    // Build and Test run first, each only once
    if err := goxt.Deps(Build, Test); err != nil {
        return err
    }
    return deploy()
}

func Test() error {
    // Build runs first
    if err := goxt.Deps(Build); err != nil {
        return err
    }
    return runTests()
}

func Build() error {
    return compile()
}
```

Running `goxt deploy` executes: Build → Test → Deploy (Build runs only once).

## Task Parameters

### Positional args

```go
// Greet prints a greeting. Usage: goxt greet <name>
func Greet() error {
    if len(os.Args) < 3 {
        return fmt.Errorf("usage: goxt greet <name>")
    }
    fmt.Printf("Hello, %s!\n", os.Args[2])
    return nil
}
```

### Flag-based args

```go
// Test runs tests. Flags: -v verbose, -race race detector
func Test() error {
    fs := flag.NewFlagSet("test", flag.ExitOnError)
    verbose := fs.Bool("v", false, "verbose")
    race := fs.Bool("race", false, "race detector")
    fs.Parse(os.Args[2:])

    args := []string{"test"}
    if *verbose { args = append(args, "-v") }
    if *race { args = append(args, "-race") }
    args = append(args, "./...")

    cmd := exec.Command("go", args...)
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    return cmd.Run()
}
```

## Conventions

| Element | Convention |
|---------|------------|
| Task function | Exported, `func() error` |
| Task name | CamelCase → kebab-case (`RunTests` → `run-tests`) |
| Description | First line of godoc comment |

## Comparison

| Tool | Dependencies | Task Deps | `go run` support |
|------|--------------|-----------|------------------|
| **goxt (Mode A)** | Zero | ✗ | ✗ |
| **goxt (Mode B)** | One | ✓ | ✓ |
| [Mage](https://magefile.org/) | Binary | ✓ | ✗ |
| [Task](https://taskfile.dev/) | Binary | ✓ | ✗ |
| Make | System | ✓ | ✗ |

## Why goxt?

- **Flexible**: Zero deps or full features - your choice
- **Just Go**: No YAML, no DSL, no magic
- **IDE support**: Full autocomplete, refactoring, debugging
- **No binary required**: Mode A works with just the CLI, Mode B works with `go run`

## License

MIT
