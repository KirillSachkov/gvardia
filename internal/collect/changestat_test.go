package collect

import (
	"context"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestParseNumstat(t *testing.T) {
	got := parseNumstat([]byte("3\t1\tfile.go\n120\t0\tui/Thread.tsx\n-\t-\timg.png\n"))
	want := model.ChangeStat{Files: 3, Added: 123, Removed: 1}
	if got != want {
		t.Errorf("parseNumstat = %+v, want %+v", got, want)
	}
}

type numstatRunner struct{ out string }

func (r numstatRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return []byte(r.out), nil
}

func TestChangeStatFor(t *testing.T) {
	st := ChangeStatFor(context.Background(), numstatRunner{out: "5\t2\ta.go\n"}, "/wt", "main")
	if st != (model.ChangeStat{Files: 1, Added: 5, Removed: 2}) {
		t.Errorf("ChangeStatFor = %+v", st)
	}
}
