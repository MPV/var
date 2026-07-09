package varcore

import (
	"strings"
	"testing"
)

func TestHashSourceDeterministic(t *testing.T) {
	if HashSource("abc") != HashSource("abc") {
		t.Error("hash not deterministic")
	}
}

func TestHashSourceChangesForOneCharDifference(t *testing.T) {
	if HashSource("abc") == HashSource("abd") {
		t.Error("hash collision on one-character difference")
	}
}

func TestHashSourceNamespacedPrefix(t *testing.T) {
	if !strings.HasPrefix(HashSource("abc"), "fnv1a:") {
		t.Error("hash not namespaced with fnv1a: prefix")
	}
}

// TestHashSourceMatchesTypeScriptVectors pins the algorithm to the exact vectors
// shared by every port (typescript/.../tests/hash.test.ts).
func TestHashSourceMatchesTypeScriptVectors(t *testing.T) {
	cases := map[string]string{
		"hello":     "fnv1a:4f9f2cab",
		"abc":       "fnv1a:1a47e90b",
		"# Title\n": "fnv1a:4eace75e",
	}
	for in, want := range cases {
		if got := HashSource(in); got != want {
			t.Errorf("HashSource(%q) = %q, want %q", in, got, want)
		}
	}
}
