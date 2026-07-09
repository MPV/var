package varrunner

import "testing"

func TestGlobMatching(t *testing.T) {
	cases := []struct {
		glob, path string
		want       bool
	}{
		{"**/*.md", "a.md", true},
		{"**/*.md", "specs/deep/a.md", true},
		{"**/*.md", "a.ts", false},
		{"specs/**/*.md", "specs/a.md", true},
		{"specs/**/*.md", "specs/sub/a.md", true},
		{"specs/**/*.md", "other/a.md", false},
		{"*.md", "a.md", true},
		{"*.md", "sub/a.md", false},
	}
	for _, c := range cases {
		if got := matchesAny(c.path, []string{c.glob}); got != c.want {
			t.Errorf("glob %q vs %q = %v, want %v", c.glob, c.path, got, c.want)
		}
	}
}

func TestMatchSpecHonoursExclude(t *testing.T) {
	if !MatchSpec("/r/specs/a.md", []string{"**/*.md"}, nil, "/r") {
		t.Error("expected include match")
	}
	if MatchSpec("/r/specs/wip/a.md", []string{"**/*.md"}, []string{"specs/wip/**"}, "/r") {
		t.Error("expected exclude to win")
	}
}
