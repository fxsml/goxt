// goxt - Minimal task runner for Go projects.
//
// Usage:
//
//	goxt --init          Create a new goxt.go file (Mode A: zero deps)
//	goxt --init --lib    Create goxt.go with library (Mode B: has Deps())
//	goxt <task>          Run a task
//	goxt                 Show help with task descriptions
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

// Mode A template: zero dependencies, no main needed
const templateSimple = `//go:build ignore

// goxt.go - Project tasks. Run with: goxt <task>
package main

import (
	"fmt"
	"os"
	"os/exec"
)

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

// Greet prints a greeting. Usage: goxt greet <name>
func Greet() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: goxt greet <name>")
	}
	fmt.Printf("Hello, %s!\n", os.Args[2])
	return nil
}
`

// Mode B template: with library, has Deps() and Run()
const templateLib = `//go:build ignore

// goxt.go - Project tasks. Run with: goxt <task> or go run goxt.go <task>
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fxsml/goxt"
)

func main() {
	goxt.Run(Build, Test, Deploy)
}

// Build compiles the project.
func Build() error {
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "./...")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// Test runs the tests.
func Test() error {
	fmt.Println("Testing...")
	cmd := exec.Command("go", "test", "./...")
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
`

type task struct {
	Name string
	Desc string
}

func main() {
	// Handle --init flag
	if hasFlag("--init") || hasFlag("-init") {
		useLib := hasFlag("--lib") || hasFlag("-lib")
		if err := initGoxtFile(useLib); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	goxtFile := findGoxtFile()
	if goxtFile == "" {
		fmt.Fprintln(os.Stderr, "goxt.go not found. Run 'goxt --init' to create one.")
		os.Exit(1)
	}

	tasks, hasMain, err := parseGoxtFile(goxtFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing goxt.go: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		printHelp(tasks)
		return
	}

	taskName := os.Args[1]

	// Find the task to get the original function name
	var funcName string
	for _, t := range tasks {
		if t.Name == taskName {
			funcName = kebabToCamel(t.Name)
			break
		}
	}
	if funcName == "" {
		fmt.Fprintf(os.Stderr, "unknown task: %s\n", taskName)
		os.Exit(1)
	}

	// Run the task
	dir := filepath.Dir(goxtFile)
	if hasMain {
		// Has main() - just run directly
		runGoxtFile(goxtFile, dir, os.Args[1:])
	} else {
		// No main() - generate temp wrapper and run
		runWithWrapper(goxtFile, dir, funcName, os.Args[2:])
	}
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func initGoxtFile(useLib bool) error {
	path := "goxt.go"
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("goxt.go already exists")
	}

	var template string
	if useLib {
		template = templateLib
	} else {
		template = templateSimple
	}

	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return err
	}

	fmt.Println("Created goxt.go")

	if useLib {
		// Run go get to add dependency
		fmt.Println("Adding goxt dependency...")
		cmd := exec.Command("go", "get", "github.com/fxsml/goxt")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("Note: run 'go get github.com/fxsml/goxt' to add the dependency")
		}
		fmt.Println("\nRun tasks with:")
		fmt.Println("  goxt build")
		fmt.Println("  go run goxt.go build")
	} else {
		fmt.Println("\nRun tasks with:")
		fmt.Println("  goxt build")
	}

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

func parseGoxtFile(filename string) ([]task, bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}

	// Check if main() exists
	hasMain := false
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "main" && fn.Recv == nil {
				hasMain = true
				break
			}
		}
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
	return tasks, hasMain, nil
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

func kebabToCamel(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '-' {
			upper = true
		} else {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func printHelp(tasks []task) {
	fmt.Println("Usage: goxt <task> [args...]")
	fmt.Println("\nFlags:")
	fmt.Println("  --init       Create a new goxt.go file (zero deps)")
	fmt.Println("  --init --lib Create goxt.go with library support (has Deps())")
	fmt.Println("  --help       Show this help")
	fmt.Println("\nTasks:")

	if len(tasks) == 0 {
		fmt.Println("  (no tasks found)")
		return
	}

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

func runGoxtFile(goxtFile, dir string, args []string) {
	cmdArgs := append([]string{"run", goxtFile}, args...)
	cmd := exec.Command("go", cmdArgs...)
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

func runWithWrapper(goxtFile, dir, funcName string, args []string) {
	// Read original file
	content, err := os.ReadFile(goxtFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading goxt.go: %v\n", err)
		os.Exit(1)
	}

	// Remove //go:build ignore line so the temp file can be compiled
	contentStr := string(content)
	contentStr = strings.Replace(contentStr, "//go:build ignore\n", "", 1)
	contentStr = strings.Replace(contentStr, "//go:build ignore\r\n", "", 1)

	// Generate main() wrapper
	// Inject task name into os.Args so task functions see consistent arg positions
	// (os.Args[1] = task name, os.Args[2:] = task args)
	taskKebab := camelToKebab(funcName)
	wrapper := fmt.Sprintf(`

// Auto-generated main for goxt CLI
func main() {
	// Inject task name so os.Args positions match normal usage
	os.Args = append([]string{os.Args[0], %q}, os.Args[1:]...)
	if err := %s(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, taskKebab, funcName)

	// Create temp file in same directory (for relative imports)
	// Note: filename must not start with . or _ (Go ignores those)
	tmpFile := filepath.Join(dir, "goxt_tmp_.go")
	if err := os.WriteFile(tmpFile, []byte(contentStr+wrapper), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	// Run the temp file
	cmdArgs := append([]string{"run", tmpFile}, args...)
	cmd := exec.Command("go", cmdArgs...)
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
