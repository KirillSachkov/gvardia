package collect

import "testing"

func TestTaskFromBranch(t *testing.T) {
	cases := []struct {
		branch string
		want   string
	}{
		{"feat/675-s3", "#675"},
		{"codex/599-global-full-access", "#599"},
		{"675-x", "#675"},
		{"AUTH-12-login", "AUTH-12"},
		{"OBS-3", "OBS-3"},
		{"fix-#42", "#42"},
		{"dev", ""},
		{"main", ""},
		{"fix/quiz-render", ""},
		{"backmerge-dev", ""},
	}
	for _, c := range cases {
		if got := TaskFromBranch(c.branch); got != c.want {
			t.Errorf("TaskFromBranch(%q) = %q, want %q", c.branch, got, c.want)
		}
	}
}
