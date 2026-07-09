// Package goconformance gates the Go port against the shared conformance corpus
// for the stages that need a real step-definition fixture per bundle (registry,
// plan, trace). It lives in the conformance module (joined to the go/ module by
// the repo-root go.work) so it can import each bundle's fixture package; the
// pure-parse (var-doc) stage is gated inside go/varcore with no fixtures.
package goconformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oselvar/var/go/oselvar"
	"github.com/oselvar/var/go/varcore"

	bundle01 "github.com/oselvar/var/conformance/bundles/01-roman-numerals"
	bundle02 "github.com/oselvar/var/conformance/bundles/02-context-isolation"
	bundle03 "github.com/oselvar/var/conformance/bundles/03-expected-failure"
	bundle04 "github.com/oselvar/var/conformance/bundles/04-tables-and-docstrings"
	bundle05 "github.com/oselvar/var/conformance/bundles/05-ambiguous-match"
	bundle06 "github.com/oselvar/var/conformance/bundles/06-doc-string-mismatch"
	bundle07 "github.com/oselvar/var/conformance/bundles/07-row-check-mismatch"
	bundle08 "github.com/oselvar/var/conformance/bundles/08-string-capture"
	bundle09 "github.com/oselvar/var/conformance/bundles/09-expected-message-mismatch"
	bundle10 "github.com/oselvar/var/conformance/bundles/10-error-fence-without-step"
	bundle11 "github.com/oselvar/var/conformance/bundles/11-emoji-offsets"
	bundle12 "github.com/oselvar/var/conformance/bundles/12-combining-marks"
	bundle13 "github.com/oselvar/var/conformance/bundles/13-custom-parameter-type"
	bundle14 "github.com/oselvar/var/conformance/bundles/14-stateless-steps"
	bundle15 "github.com/oselvar/var/conformance/bundles/15-custom-parameter-format"
)

// bundleFactories maps a bundle directory name to its fixture constructor. An
// explicit map (rather than reflective loading) keeps the mapping compiler-checked,
// mirroring the Java port's loadFixture switch.
var bundleFactories = map[string]func() *oselvar.Bundle{
	"01-roman-numerals":            bundle01.Bundle,
	"02-context-isolation":         bundle02.Bundle,
	"03-expected-failure":          bundle03.Bundle,
	"04-tables-and-docstrings":     bundle04.Bundle,
	"05-ambiguous-match":           bundle05.Bundle,
	"06-doc-string-mismatch":       bundle06.Bundle,
	"07-row-check-mismatch":        bundle07.Bundle,
	"08-string-capture":            bundle08.Bundle,
	"09-expected-message-mismatch": bundle09.Bundle,
	"10-error-fence-without-step":  bundle10.Bundle,
	"11-emoji-offsets":             bundle11.Bundle,
	"12-combining-marks":           bundle12.Bundle,
	"13-custom-parameter-type":     bundle13.Bundle,
	"14-stateless-steps":           bundle14.Bundle,
	"15-custom-parameter-format":   bundle15.Bundle,
}

func TestRegistryConformance(t *testing.T) {
	root := repoRoot(t)
	for name, factory := range bundleFactories {
		t.Run(name, func(t *testing.T) {
			bundle := factory()
			got, err := varcore.CanonicalStringify(varcore.ToRegistryArtifact(bundle.Registry))
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join(root, "conformance", "bundles", name, "golden", "registry.json")
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("registry mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

// repoRoot walks up to the monorepo root (the dir containing conformance/bundles).
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
			t.Fatal("could not locate repo root")
		}
		dir = parent
	}
}
