// goxt - Enhanced task runner that extracts help from goxt.go godoc comments.
//
// Usage:
//
//	goxt init       Create a new goxt.go file
//	goxt <task>     Run a task
//	goxt           Show help with descriptions
package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const goxtTemplate = `//go:build ignore

// goxt.go - Project tasks. Run: go run goxt.go <task>
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	tasks := map[string]func() error{
		"test":  Test,
		"build": Build,
		"greet": Greet,
	}
	if len(os.Args) < 2 {
		fmt.Println("Tasks:")
		for t := range tasks { fmt.Println(" ", t) }
		return
	}
	if fn, ok := tasks[os.Args[1]]; ok {
		if err := fn(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	} else {
		fmt.Println("unknown task:", os.Args[1])
		os.Exit(1)
	}
}

// Test runs tests. Accepts: -v for verbose, -race for race detection.
func Test() error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose output")
	race := fs.Bool("race", false, "enable race detector")
	fs.Parse(os.Args[2:])

	args := []string{"test"}
	if *verbose { args = append(args, "-v") }
	if *race { args = append(args, "-race") }
	args = append(args, "./...")

	cmd := exec.Command("go", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// Build compiles the project.
func Build() error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// Greet prints a greeting. Usage: greet <name>
func Greet() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: greet <name>")
	}
	fmt.Printf("Hello, %s!\n", os.Args[2])
	return nil
}
`

type task struct {
	Name string
	Desc string
}

func main() {
	// Handle init before looking for goxt.go
	if len(os.Args) >= 2 && os.Args[1] == "init" {
		if err := initGoxtFile(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	goxtFile := findGoxtFile()
	if goxtFile == "" {
		fmt.Fprintln(os.Stderr, "goxt.go not found. Run 'goxt init' to create one.")
		os.Exit(1)
	}

	tasks, err := parseTasks(goxtFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing goxt.go: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		printHelp(tasks)
		return
	}

	// Run the task via go run, passing all args
	dir := filepath.Dir(goxtFile)
	args := append([]string{"run", goxtFile}, os.Args[1:]...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func initGoxtFile() error {
	path := "goxt.go"
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("goxt.go already exists")
	}
	if err := os.WriteFile(path, []byte(goxtTemplate), 0644); err != nil {
		return err
	}
	fmt.Println("Created goxt.go")
	fmt.Println("\nRun tasks with:")
	fmt.Println("  go run goxt.go test")
	fmt.Println("  goxt test")
	return nil
}

func findGoxtFile() string {
	dir, _ := os.Getwd()
	for {
		path := filepath.Join(dir, "goxt.go")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func parseTasks(filename string) ([]task, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pkg := &ast.Package{
		Name:  "main",
		Files: map[string]*ast.File{filename: f},
	}
	d := doc.New(pkg, "", doc.AllDecls)

	var tasks []task
	for _, fn := range d.Funcs {
		if !ast.IsExported(fn.Name) {
			continue
		}
		if !isTaskFunc(fn.Decl) {
			continue
		}
		if fn.Name == "Tasks" {
			continue
		}

		tasks = append(tasks, task{
			Name: camelToKebab(fn.Name),
			Desc: formatDoc(fn.Doc, fn.Name),
		})
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})
	return tasks, nil
}

func isTaskFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil {
		return false
	}
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	ret := fn.Type.Results.List[0]
	ident, ok := ret.Type.(*ast.Ident)
	return ok && ident.Name == "error"
}

func formatDoc(docStr, funcName string) string {
	if docStr == "" {
		return ""
	}
	text := strings.TrimSpace(docStr)
	if idx := strings.Index(text, "\n"); idx > 0 {
		text = text[:idx]
	}
	if strings.HasPrefix(text, funcName+" ") {
		text = text[len(funcName)+1:]
	}
	if len(text) > 0 {
		text = strings.ToUpper(text[:1]) + text[1:]
	}
	return text
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

func printHelp(tasks []task) {
	fmt.Println("Usage: goxt <task> [args...]")
	fmt.Println("\nCommands:")
	fmt.Println("  init    Create a new goxt.go file")
	fmt.Println("  help    Show this help")
	fmt.Println("\nTasks:")

	maxLen := 0
	for _, t := range tasks {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}
	for _, t := range tasks {
		if t.Desc != "" {
			fmt.Printf("  %-*s  %s\n", maxLen, t.Name, t.Desc)
		} else {
			fmt.Printf("  %s\n", t.Name)
		}
	}
}
