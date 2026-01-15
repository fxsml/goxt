// Package goxt provides a minimal task runner for Go projects.
//
// Use Run() to register and execute tasks:
//
//	func main() {
//	    goxt.Run(Build, Test, Deploy)
//	}
//
// Use Deps() to declare task dependencies:
//
//	func Deploy() error {
//	    return goxt.Deps(Build, Test)
//	}
package goxt

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unicode"
)

// Run registers tasks and executes based on CLI args.
// Task names are derived from function names (BuildApp → "build-app").
// Handles help display, error reporting, and exit codes.
func Run(tasks ...func() error) {
	taskMap := make(map[string]func() error)
	var names []string

	for _, fn := range tasks {
		name := camelToKebab(funcName(fn))
		taskMap[name] = fn
		names = append(names, name)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run goxt.go <task> [args...]")
		fmt.Println("\nTasks:")
		for _, name := range names {
			fmt.Println(" ", name)
		}
		return
	}

	taskName := os.Args[1]
	if taskName == "help" || taskName == "-h" || taskName == "--help" {
		fmt.Println("Usage: go run goxt.go <task> [args...]")
		fmt.Println("\nTasks:")
		for _, name := range names {
			fmt.Println(" ", name)
		}
		return
	}

	fn, ok := taskMap[taskName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown task: %s\n", taskName)
		fmt.Fprintln(os.Stderr, "\nAvailable tasks:")
		for _, name := range names {
			fmt.Fprintln(os.Stderr, " ", name)
		}
		os.Exit(1)
	}

	if err := fn(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Deps runs dependency tasks exactly once.
// Safe for diamond dependencies - each task runs only once per execution.
func Deps(tasks ...func() error) error {
	for _, fn := range tasks {
		if err := runOnce(fn); err != nil {
			return err
		}
	}
	return nil
}

var (
	onceMu sync.Mutex
	onces  = make(map[uintptr]*onceResult)
)

type onceResult struct {
	once sync.Once
	err  error
}

func runOnce(fn func() error) error {
	ptr := reflect.ValueOf(fn).Pointer()

	onceMu.Lock()
	res, ok := onces[ptr]
	if !ok {
		res = &onceResult{}
		onces[ptr] = res
	}
	onceMu.Unlock()

	res.once.Do(func() {
		res.err = fn()
	})
	return res.err
}

func funcName(fn func() error) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

func camelToKebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
