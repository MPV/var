// Package varrunner is the imperative shell of the Go port: spec discovery,
// step loading, planning, failure rendering, and the filesystem drift baseline.
// Port of var_runner (python) / var-runner. It performs I/O; the pipeline logic
// lives in varcore.
package varrunner

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// globToRegex translates a glob with **, *, ? into a compiled regex. Port of
// _glob_to_regex — same semantics as pathlib full_match / PEP 428.
func globToRegex(pattern string) *regexp.Regexp {
	var b strings.Builder
	i, n := 0, len(pattern)
	for i < n {
		switch {
		case pattern[i] == '/' && strings.HasPrefix(pattern[i:], "/**/"):
			b.WriteString("/(?:.+/)?")
			i += 4
		case pattern[i] == '/' && strings.HasPrefix(pattern[i:], "/**") && i+3 == n:
			b.WriteString("(?:/.*)?")
			i += 3
		case pattern[i] == '*' && strings.HasPrefix(pattern[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 3
		case pattern[i] == '*' && strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i += 2
		case pattern[i] == '*':
			b.WriteString("[^/]*")
			i++
		case pattern[i] == '?':
			b.WriteString("[^/]")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	return regexp.MustCompile("^(?:" + b.String() + ")$")
}

func matchesAny(rel string, globs []string) bool {
	for _, g := range globs {
		if globToRegex(g).MatchString(rel) {
			return true
		}
	}
	return false
}

// relPosix returns path relative to root with forward slashes.
func relPosix(path, root string) string {
	absPath, _ := filepath.Abs(path)
	absRoot, _ := filepath.Abs(root)
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		rel = path
	}
	return filepath.ToSlash(rel)
}

// MatchSpec reports whether path matches an include glob and no exclude glob.
func MatchSpec(path string, include, exclude []string, root string) bool {
	rel := relPosix(path, root)
	return matchesAny(rel, include) && !matchesAny(rel, exclude)
}

// FindSpecs returns existing files under root matching any include glob minus any
// exclude, sorted. Port of find_specs.
func FindSpecs(include, exclude []string, root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := relPosix(path, root)
		if matchesAny(rel, include) && !matchesAny(rel, exclude) {
			seen[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}
