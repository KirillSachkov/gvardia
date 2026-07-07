package collect

import "testing"

func TestParseStatus(t *testing.T) {
	tests := []struct {
		fixture string
		want    status
	}{
		{"status_clean.txt", status{HasCommits: true}},
		{"status_dirty.txt", status{Dirty: true, HasCommits: true}},
		{"status_ahead_behind.txt", status{Ahead: 6, Behind: 2, HasCommits: true}},
		{"status_initial.txt", status{HasCommits: false}},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			got := parseStatus(readFixture(t, tt.fixture))
			if got != tt.want {
				t.Errorf("parseStatus(%s) = %+v, want %+v", tt.fixture, got, tt.want)
			}
		})
	}
}
