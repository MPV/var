// Package varconfig reads var.config.json — the single source of truth for what
// is a spec and where steps/snippets live, shared verbatim across every port.
// Port of var_config (python) / config.ts. Pure: it parses one file and fails
// loud on anything malformed; it performs no other I/O and does not serialize
// (the conformance harness projects Artifact() through the shared canonical
// writer).
package varconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VarConfig is the parsed configuration.
type VarConfig struct {
	DocsInclude    []string
	DocsExclude    []string
	Steps          []string
	Snippets       map[string]string
	ScannerPlugins []string
}

var knownKeys = map[string]bool{"$schema": true, "docs": true, "steps": true, "snippets": true, "scannerPlugins": true}
var knownDocsKeys = map[string]bool{"include": true, "exclude": true}

// ReadVarConfig reads <root>/var.config.json. A missing file yields an empty
// config (tools no-op). Malformed JSON, wrong types, or unknown keys return an
// error beginning with the file path — a typo'd config must fail loud.
func ReadVarConfig(root string) (VarConfig, error) {
	path := filepath.Join(root, "var.config.json")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return VarConfig{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return VarConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	var top any
	if err := json.Unmarshal(raw, &top); err != nil {
		return VarConfig{}, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	data, ok := top.(map[string]any)
	if !ok {
		return VarConfig{}, fmt.Errorf("%s: top level must be an object", path)
	}
	if unknown := unknownKeys(data, knownKeys); len(unknown) > 0 {
		return VarConfig{}, fmt.Errorf("%s: unknown key(s): %s", path, strings.Join(unknown, ", "))
	}

	docs := map[string]any{}
	if d, present := data["docs"]; present && d != nil {
		m, ok := d.(map[string]any)
		if !ok {
			return VarConfig{}, fmt.Errorf("%s: 'docs' must be an object", path)
		}
		docs = m
	}
	if unknown := unknownKeys(docs, knownDocsKeys); len(unknown) > 0 {
		return VarConfig{}, fmt.Errorf("%s: unknown docs key(s): %s", path, strings.Join(unknown, ", "))
	}

	snippets := map[string]string{}
	if s, present := data["snippets"]; present && s != nil {
		m, ok := s.(map[string]any)
		if !ok {
			return VarConfig{}, fmt.Errorf("%s: 'snippets' must be an object of strings", path)
		}
		for k, v := range m {
			str, ok := v.(string)
			if !ok {
				return VarConfig{}, fmt.Errorf("%s: 'snippets' must be an object of strings", path)
			}
			snippets[k] = str
		}
	}

	include, err := stringSlice(docs["include"], "docs.include", path)
	if err != nil {
		return VarConfig{}, err
	}
	exclude, err := stringSlice(docs["exclude"], "docs.exclude", path)
	if err != nil {
		return VarConfig{}, err
	}
	steps, err := stringSlice(data["steps"], "steps", path)
	if err != nil {
		return VarConfig{}, err
	}
	plugins, err := stringSlice(data["scannerPlugins"], "scannerPlugins", path)
	if err != nil {
		return VarConfig{}, err
	}

	return VarConfig{
		DocsInclude:    include,
		DocsExclude:    exclude,
		Steps:          steps,
		Snippets:       snippets,
		ScannerPlugins: plugins,
	}, nil
}

func unknownKeys(m map[string]any, known map[string]bool) []string {
	var out []string
	for k := range m {
		if !known[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func stringSlice(value any, key, path string) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	arr, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: '%s' must be an array of strings", path, key)
	}
	out := make([]string, len(arr))
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s: '%s' must be an array of strings", path, key)
		}
		out[i] = s
	}
	return out, nil
}

// Artifact projects the config to the shared conformance wire shape (canonical
// keys; empty collections present). Pure — the caller serializes it.
func (c VarConfig) Artifact() map[string]any {
	toAny := func(ss []string) []any {
		out := make([]any, len(ss))
		for i, s := range ss {
			out[i] = s
		}
		return out
	}
	snippets := map[string]any{}
	for k, v := range c.Snippets {
		snippets[k] = v
	}
	return map[string]any{
		"docs": map[string]any{
			"include": toAny(c.DocsInclude),
			"exclude": toAny(c.DocsExclude),
		},
		"steps":          toAny(c.Steps),
		"snippets":       snippets,
		"scannerPlugins": toAny(c.ScannerPlugins),
	}
}
