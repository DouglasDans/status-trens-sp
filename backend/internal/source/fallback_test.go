package source

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeSource struct {
	snap  Snapshot
	err   error
	calls int
}

func (f *fakeSource) Fetch(context.Context) (Snapshot, error) {
	f.calls++
	return f.snap, f.err
}

func TestFallbackUsesFirstSourceThatSucceeds(t *testing.T) {
	primary := &fakeSource{snap: Snapshot{Source: "primary"}}
	secondary := &fakeSource{snap: Snapshot{Source: "secondary"}}

	snap, err := Fallback{primary, secondary}.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "primary" || secondary.calls != 0 {
		t.Errorf("source = %q, secondary calls = %d; want primary and 0", snap.Source, secondary.calls)
	}
}

func TestFallbackTriesNextSourceOnError(t *testing.T) {
	primary := &fakeSource{err: errors.New("primary down")}
	secondary := &fakeSource{snap: Snapshot{Source: "secondary"}}

	snap, err := Fallback{primary, secondary}.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "secondary" {
		t.Errorf("source = %q, want secondary", snap.Source)
	}
}

func TestFallbackReportsAllErrorsWhenEverySourceFails(t *testing.T) {
	primary := &fakeSource{err: errors.New("primary down")}
	secondary := &fakeSource{err: errors.New("secondary down")}

	_, err := Fallback{primary, secondary}.Fetch(context.Background())
	if err == nil {
		t.Fatal("want error, got nil")
	}
	for _, msg := range []string{"primary down", "secondary down"} {
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("error %q does not mention %q", err, msg)
		}
	}
}
