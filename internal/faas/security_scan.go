package faas

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// SecurityScanResult contains the findings of a security scan
type SecurityScanResult struct {
	Passed     bool
	Violations []Violation
	Warnings   []Warning
	Summary    string
}

// Violation represents a critical security issue that blocks execution
type Violation struct {
	Code    string
	Message string
	Line    int
	Column  int
	Detail  string
}

// Warning represents a non-critical issue or best practice suggestion
type Warning struct {
	Code    string
	Message string
	Line    int
	Column  int
	Detail  string
}

// ForbiddenPatterns defines regex patterns for dangerous function calls
var ForbiddenPatterns = []struct {
	Pattern *regexp.Regexp
	Code    string
	Message string
}{
	{
		regexp.MustCompile(`\bos\s*\.\s*Exec\s*\(`),
		"SEC001",
		"Direct OS command execution is forbidden",
	},
	{
		regexp.MustCompile(`\bos\s*\.\s*StartProcess\s*\(`),
		"SEC002",
		"Process spawning is forbidden",
	},
	{
		regexp.MustCompile(`\bexec\s*\.\s*Command\s*\(`),
		"SEC003",
		"Command execution is forbidden",
	},
	{
		regexp.MustCompile(`\bexec\s*\.\s*LookPath\s*\(`),
		"SEC004",
		"Path lookup is forbidden",
	},
	{
		regexp.MustCompile(`\bhttp\s*\.\s*Client\s*\{\s*Transport\s*:\s*\&\s*http\s*\.\s*Transport\s*\{[^}]*DisableCompression\s*:\s*true`),
		"SEC005",
		"Disabling compression might be used for evasion",
	},
	{
		regexp.MustCompile(`\bsocket\s*\.\s*Connect\s*\(`),
		"SEC006",
		"Raw socket connections are forbidden",
	},
	{
		regexp.MustCompile(`\bunsafe\s*\.\s*Pointer`),
		"SEC007",
		"Unsafe pointer manipulation is forbidden",
	},
	{
		regexp.MustCompile(`\bsyscall\s*\.\s*Syscall`),
		"SEC008",
		"Direct system calls are forbidden",
	},
}

// Scanner performs static analysis on Go source code
type Scanner struct {
	// Additional forbidden packages
	ForbiddenPackages []string
	// Allowed standard library packages (subset)
	AllowedStdLib []string
}

// NewScanner creates a new security scanner
func NewScanner() *Scanner {
	return &Scanner{
		ForbiddenPackages: []string{
			"os/exec",
			"os/start",
			"syscall",
			"unsafe",
			"net/http/httputil", // Reverse proxy risks
			"reflect",           // Can be used to bypass type safety
			"runtime",           // Can expose runtime internals
		},
		AllowedStdLib: []string{
			"bytes", "context", "encoding", "errors", "fmt", "io",
			"log", "net/http", "path/filepath", "sort", "strconv",
			"strings", "sync", "time", "unicode", "math", "math/rand",
		},
	}
}

// Scan analyzes the source code for security issues
func (s *Scanner) Scan(sourceCode string) (*SecurityScanResult, error) {
	result := &SecurityScanResult{
		Passed:     true,
		Violations: []Violation{},
		Warnings:   []Warning{},
	}

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "user_code.go", sourceCode, parser.ParseComments)
	if err != nil {
		// If parsing fails, it might be a syntax error, which is caught by the compiler.
		// But we can still try to scan the raw text for obvious patterns.
		result.Warnings = append(result.Warnings, Warning{
			Code:    "PARSE001",
			Message: "Failed to parse AST, falling back to regex scan",
			Detail:  err.Error(),
		})
		return s.scanRawText(sourceCode, result), nil
	}

	// 1. Check Imports
	s.checkImports(file, result)

	// 2. Check Function Calls (AST)
	s.checkCalls(file, result, fset)

	// 3. Check for forbidden patterns in comments/strings (Regex fallback)
	s.checkRawPatterns(sourceCode, result)

	// Determine overall pass/fail
	if len(result.Violations) > 0 {
		result.Passed = false
		result.Summary = fmt.Sprintf("Scan failed with %d violations and %d warnings", len(result.Violations), len(result.Warnings))
	} else if len(result.Warnings) > 0 {
		result.Summary = fmt.Sprintf("Scan passed with %d warnings", len(result.Warnings))
	} else {
		result.Summary = "Scan passed with no issues"
	}

	return result, nil
}

// checkImports verifies that only allowed packages are imported
func (s *Scanner) checkImports(file *ast.File, result *SecurityScanResult) {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")

		// Check forbidden packages
		for _, forbidden := range s.ForbiddenPackages {
			if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
				result.Violations = append(result.Violations, Violation{
					Code:    "IMP001",
					Message: fmt.Sprintf("Forbidden package imported: %s", path),
					Line:    fset.Position(imp.Pos()).Line,
					Column:  fset.Position(imp.Pos()).Column,
					Detail:  fmt.Sprintf("Package '%s' is not allowed in user functions.", path),
				})
			}
		}

		// Check if it's a standard library or allowed
		// Note: We rely on the BuildEnv for strict whitelist, but this is a secondary check.
		isAllowed := false
		for _, allowed := range s.AllowedStdLib {
			if path == allowed || strings.HasPrefix(path, allowed+"/") {
				isAllowed = true
				break
			}
		}

		// Allow 'song' package
		if path == "song" || strings.HasPrefix(path, "song/") {
			isAllowed = true
		}

		// If not allowed and not standard (no dot in root), flag it
		if !isAllowed && !strings.Contains(path, ".") {
			// Might be a standard lib we missed, or a custom one.
			// BuildEnv will catch the custom one, but we warn here.
			result.Warnings = append(result.Warnings, Warning{
				Code:    "IMP002",
				Message: fmt.Sprintf("Unknown package imported: %s", path),
				Line:    fset.Position(imp.Pos()).Line,
				Detail:  "This package is not in the known standard library list. Build will verify.",
			})
		}
	}
}

// checkCalls inspects function calls for dangerous operations
func (s *Scanner) checkCalls(file *ast.File, result *SecurityScanResult, fset *token.FileSet) {
	ast.Inspect(file, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check for os/exec, syscall, unsafe calls
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				pkgName := ident.Name
				funcName := selExpr.Sel.Name

				// Dangerous combinations
				if pkgName == "os" && (funcName == "Exec" || funcName == "StartProcess") {
					result.Violations = append(result.Violations, Violation{
						Code:    "CALL001",
						Message: fmt.Sprintf("Dangerous function call: os.%s", funcName),
						Line:    fset.Position(callExpr.Pos()).Line,
						Column:  fset.Position(callExpr.Pos()).Column,
						Detail:  "Executing OS commands is strictly forbidden.",
					})
				}
				if pkgName == "syscall" {
					result.Violations = append(result.Violations, Violation{
						Code:    "CALL002",
						Message: "Direct syscall usage is forbidden",
						Line:    fset.Position(callExpr.Pos()).Line,
						Column:  fset.Position(callExpr.Pos()).Column,
						Detail:  "Syscalls bypass the Go runtime safety checks.",
					})
				}
				if pkgName == "unsafe" {
					result.Violations = append(result.Violations, Violation{
						Code:    "CALL003",
						Message: "Unsafe package usage is forbidden",
						Line:    fset.Position(callExpr.Pos()).Line,
						Column:  fset.Position(callExpr.Pos()).Column,
						Detail:  "The unsafe package breaks type safety.",
					})
				}
			}
		}
		return true
	})
}

// checkRawPatterns scans the raw source text for patterns the AST might miss
func (s *Scanner) checkRawPatterns(sourceCode string, result *SecurityScanResult) {
	lines := strings.Split(sourceCode, "\n")
	for i, line := range lines {
		for _, fp := range ForbiddenPatterns {
			if fp.Pattern.MatchString(line) {
				// Avoid duplicates if AST already caught it
				alreadyFound := false
				for _, v := range result.Violations {
					if v.Code == fp.Code && v.Line == i+1 {
						alreadyFound = true
						break
					}
				}
				if !alreadyFound {
					result.Violations = append(result.Violations, Violation{
						Code:    fp.Code,
						Message: fp.Message,
						Line:    i + 1,
						Column:  0,
						Detail:  "Pattern matched in source code.",
					})
				}
			}
		}
	}
}

// scanRawText is a fallback if AST parsing fails
func (s *Scanner) scanRawText(sourceCode string, result *SecurityScanResult) *SecurityScanResult {
	s.checkRawPatterns(sourceCode, result)
	return result
}

// Report generates a human-readable report of the scan results
func (r *SecurityScanResult) Report() string {
	var sb strings.Builder
	sb.WriteString("=== Security Scan Report ===\n")
	sb.WriteString(r.Summary + "\n\n")

	if len(r.Violations) > 0 {
		sb.WriteString("VIOLATIONS:\n")
		for _, v := range r.Violations {
			sb.WriteString(fmt.Sprintf("  [Line %d] %s (%s): %s\n", v.Line, v.Code, v.Message, v.Detail))
		}
		sb.WriteString("\n")
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("WARNINGS:\n")
		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  [Line %d] %s (%s): %s\n", w.Line, w.Code, w.Message, w.Detail))
		}
	}

	return sb.String()
}
