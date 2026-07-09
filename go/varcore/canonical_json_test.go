package varcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalStringifySortsKeysAndIndents(t *testing.T) {
	got, err := CanonicalStringify(map[string]any{
		"b": 1,
		"a": map[string]any{"y": 2, "x": 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": {\n    \"x\": 3,\n    \"y\": 2\n  },\n  \"b\": 1\n}\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestCanonicalStringifyEmptyCollections(t *testing.T) {
	got, err := CanonicalStringify(map[string]any{"items": []any{}, "meta": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"items\": [],\n  \"meta\": {}\n}\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestCanonicalStringifyNonASCIIRaw(t *testing.T) {
	got, err := CanonicalStringify(map[string]any{"text": "café 😀 <b>&"})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"text\": \"café 😀 <b>&\"\n}\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// TestCanonicalStringifyRoundTripsGoldens proves the writer against the real
// corpus before the pipeline exists: every committed golden is already
// canonical, so decoding one and re-serializing it must reproduce it
// byte-for-byte. Covers every bundle artifact plus the config cases.
func TestCanonicalStringifyRoundTripsGoldens(t *testing.T) {
	goldens := findGoldens(t)
	if len(goldens) == 0 {
		t.Fatal("no golden files found; is the conformance corpus present?")
	}
	for _, path := range goldens {
		t.Run(relToRepo(t, path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatalf("unmarshal %s: %v", path, err)
			}
			got, err := CanonicalStringify(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(raw) {
				t.Errorf("re-serialized %s does not match golden byte-for-byte\ngot:\n%s\nwant:\n%s", path, got, raw)
			}
		})
	}
}

// findGoldens returns every *.json golden under the conformance corpus.
func findGoldens(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var out []string
	for _, glob := range []string{
		filepath.Join(root, "conformance", "bundles", "*", "golden", "*.json"),
		filepath.Join(root, "conformance", "config", "cases", "*", "golden.json"),
	} {
		matches, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, matches...)
	}
	return out
}

// repoRoot walks up from the test's working directory to the monorepo root
// (the directory that contains the conformance corpus).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "conformance", "bundles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (conformance/bundles not found)")
		}
		dir = parent
	}
}

func relToRepo(t *testing.T, path string) string {
	t.Helper()
	rel, err := filepath.Rel(repoRoot(t), path)
	if err != nil {
		return path
	}
	return rel
}
