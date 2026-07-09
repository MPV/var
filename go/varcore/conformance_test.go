package varcore

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVarDocConformance gates the parse pipeline: for every bundle, parse
// example.md, project the var-doc artifact, serialize it canonically, and
// byte-compare it against golden/var-doc.json. Bundles 11-emoji-offsets and
// 12-combining-marks prove the UTF-16 offset layer.
func TestVarDocConformance(t *testing.T) {
	root := repoRoot(t)
	bundles, err := filepath.Glob(filepath.Join(root, "conformance", "bundles", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) == 0 {
		t.Fatal("no conformance bundles found")
	}

	for _, bundle := range bundles {
		name := filepath.Base(bundle)
		t.Run(name, func(t *testing.T) {
			examplePath := filepath.Join(bundle, "example.md")
			goldenPath := filepath.Join(bundle, "golden", "var-doc.json")

			source, err := os.ReadFile(examplePath)
			if err != nil {
				t.Skipf("no example.md: %v", err)
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("no golden: %v", err)
			}

			doc := Parse("example.md", string(source), nil)
			got, err := CanonicalStringify(ToVarDocArtifact(doc))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("var-doc mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}
