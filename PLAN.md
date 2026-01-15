# goxt - Project Evaluation

## Overview

**goxt** is a minimalist, zero-dependency task runner for Go projects. It provides a convention-based approach to defining and running project tasks as an alternative to Makefiles.

### Core Concept

- Define tasks as `func() error` functions in a `goxt.go` file
- Run them with `go run goxt.go <task>` (no installation required)
- Optional CLI (`goxt`) adds convenience features like auto-discovery and help text

## Comparison with Mage

| Aspect | goxt | Mage |
|--------|------|------|
| **Dependencies** | Zero (stdlib only) | Requires mage binary |
| **Installation** | Optional | Required |
| **Immediate use** | `go run goxt.go test` | Must install first |
| **Code size** | ~270 lines | Large ecosystem |
| **Code generation** | None | Generates helper files |
| **Learning curve** | Minimal | Higher (mg.Deps, namespaces, etc.) |
| **Task dependencies** | Manual | Automatic via `mg.Deps()` |
| **Parallelization** | Manual | Built-in |
| **Caching** | None | File change tracking |

## Daseinsberechtigung (Reason to Exist)

### Verdict: Yes, in a specific niche

goxt has a legitimate reason to exist, though not as a Mage replacement - rather as a simpler alternative for specific use cases.

### Strengths / Unique Value

1. **True zero-installation**: `go run goxt.go` works immediately - no `go install` needed. This is a genuine advantage for quick scripts or repos where you don't want to mandate tool installation.

2. **Radical simplicity**: 270 lines vs thousands. Easy to understand, audit, and fork.

3. **No magic**: Just plain Go functions. No special imports, no `mg.Deps()`, no code generation.

4. **Portability**: Clone any repo with `goxt.go`, run `go run goxt.go` - done.

### Weaknesses / Where Mage Wins

1. **No dependency management**: Mage's `mg.Deps()` lets you declare task dependencies and run them in parallel automatically. goxt has nothing comparable.

2. **No caching/incremental builds**: Mage tracks file changes.

3. **Less mature ecosystem**: Mage has years of community contributions.

4. **Manual task registration**: In goxt's template, you must manually maintain the `tasks` map. The CLI parses AST to auto-discover, but the goxt.go file itself requires manual registration.

## Target Use Cases

**goxt is ideal for:**
- Small projects wanting minimal ceremony
- Scripts where "just `go run`" is valuable
- Teams allergic to installing additional tools
- Situations where Mage feels like overkill

**Mage is better for:**
- Complex build pipelines with dependencies
- Large projects needing parallelization
- Teams already using Mage in other projects

## Core Innovation

The key differentiator - "tasks as a convention that works without any CLI installation via `go run`" - is genuinely useful and not something Mage offers directly.

## Recommendations for Improvement

1. **Auto-generate the task map**: Currently manual in the template; could be generated
2. **Simple dependency declaration**: Allow tasks to declare prerequisites
3. **Watch mode**: Re-run tasks on file changes
4. **Better argument parsing integration**: Streamline flag handling

## Conclusion

goxt is a valid "Makefile replacement" in the spirit of Unix simplicity. It's not trying to compete with Mage's full feature set - it's intentionally minimal. That's a legitimate design choice with real use cases. The project fills a gap for developers who want task automation without any installation ceremony.
