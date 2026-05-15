package song

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Router struct {
	handlersDir string
	routes      []Route
}

type Route struct {
	Pattern     *regexp.Regexp
	HandlerName string
	ParamNames  []string
	FilePath    string
}

func NewRouter(handlersDir string) *Router {
	r := &Router{
		handlersDir: handlersDir,
		routes:      []Route{},
	}
	r.scanHandlers()
	return r
}

func (r *Router) scanHandlers() {
	if _, err := os.Stat(r.handlersDir); os.IsNotExist(err) {
		os.MkdirAll(r.handlersDir, 0755)
		return
	}

	filepath.Walk(r.handlersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		relPath, err := filepath.Rel(r.handlersDir, path)
		if err != nil {
			return err
		}

		pattern := "/" + strings.TrimSuffix(relPath, ".go")
		pattern = strings.ReplaceAll(pattern, "\\", "/")

		regexPattern := convertToRegex(pattern)
		paramNames := extractParamNames(pattern)

		re, err := regexp.Compile("^" + regexPattern + "$")
		if err != nil {
			fmt.Printf("Invalid route pattern %s: %v\n", pattern, err)
			return nil
		}

		r.routes = append(r.routes, Route{
			Pattern:     re,
			HandlerName: strings.TrimSuffix(filepath.Base(path), ".go"),
			ParamNames:  paramNames,
			FilePath:    path,
		})
		return nil
	})
}

func convertToRegex(pattern string) string {
	escaped := regexp.QuoteMeta(pattern)
	re := regexp.MustCompile(`\\\{(.*?)\\\}`)
	escaped = re.ReplaceAllString(escaped, `([^/]+)`)
	return escaped
}

func extractParamNames(pattern string) []string {
	re := regexp.MustCompile(`\{(.*?)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)
	names := make([]string, len(matches))
	for i, match := range matches {
		if len(match) > 1 {
			names[i] = match[1]
		}
	}
	return names
}

func (r *Router) Match(path string) (string, map[string]string, error) {
	params := make(map[string]string)

	for _, route := range r.routes {
		matches := route.Pattern.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		for i, name := range route.ParamNames {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}

		return route.HandlerName, params, nil
	}

	return "", nil, fmt.Errorf("no matching route found for %s", path)
}

func (r *Router) GetRoutes() []Route {
	return r.routes
}
