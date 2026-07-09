package goconformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oselvar/var/go/varconfig"
	"github.com/oselvar/var/go/varcore"
)

// TestConfigConformance gates varconfig against the shared config corpus. A case
// with golden.json must parse and project to it byte-for-byte; a case with
// expect-error.txt must fail to load (the .txt is human-only).
func TestConfigConformance(t *testing.T) {
	root := repoRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "config", "cases", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no config cases found")
	}
	for _, dir := range cases {
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			goldenPath := filepath.Join(dir, "golden.json")
			errMarker := filepath.Join(dir, "expect-error.txt")

			if _, err := os.Stat(errMarker); err == nil {
				if _, err := varconfig.ReadVarConfig(dir); err == nil {
					t.Errorf("expected %s to fail loading, but it succeeded", name)
				}
				return
			}

			cfg, err := varconfig.ReadVarConfig(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := varcore.CanonicalStringify(cfg.Artifact())
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("config mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}
