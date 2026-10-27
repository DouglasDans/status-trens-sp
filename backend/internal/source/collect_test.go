package source

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// namedFake answers after delay, or never when block is set (it ignores the
// context, like a client without a deadline).
type namedFake struct {
	name  string
	snap  Snapshot
	err   error
	delay time.Duration
	block chan struct{}
}

func (f *namedFake) Name() string { return f.name }

func (f *namedFake) Fetch(ctx context.Context) (Snapshot, error) {
	if f.block != nil {
		<-f.block
	}
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
	snap := f.snap
	snap.Source = f.name
	return snap, f.err
}

func ok(name string, codes ...int) *namedFake {
	var readings []Reading
	for _, c := range codes {
		readings = append(readings, Reading{Code: c, State: status.Normal, Recognized: true, Label: "Operação Normal"})
	}
	return &namedFake{name: name, snap: Snapshot{Readings: readings}}
}

func collect(t *testing.T, timeout time.Duration, sources ...StatusSource) ([]Snapshot, string, time.Duration) {
	t.Helper()
	var logs bytes.Buffer
	start := time.Now()
	snaps := Collect(context.Background(), sources, timeout, slog.New(slog.NewTextHandler(&logs, nil)))
	return snaps, logs.String(), time.Since(start)
}

func names(snaps []Snapshot) string {
	var n []string
	for _, s := range snaps {
		n = append(n, s.Source)
	}
	return strings.Join(n, ",")
}

// R5.2 (collection part): one failing source leaves the others in, in order.
func TestCollectKeepsTheSourcesThatAnswered(t *testing.T) {
	broken := &namedFake{name: "cptm", err: errors.New("unexpected HTTP status 500")}

	snaps, _, _ := collect(t, time.Second, ok("proximotrem", 1), broken, ok("metro", 1), ok("cptm-novo", 10))

	if got := names(snaps); got != "proximotrem,metro,cptm-novo" {
		t.Errorf("sources = %s, want proximotrem,metro,cptm-novo", got)
	}
}

// R4.1
func TestCollectSlowSourceDoesNotDelayTheCycleBeyondItsTimeout(t *testing.T) {
	slow := &namedFake{name: "cptm", delay: 10 * time.Second}

	snaps, logs, took := collect(t, 50*time.Millisecond, ok("proximotrem", 1), slow)

	if got := names(snaps); got != "proximotrem" {
		t.Errorf("sources = %s, want proximotrem", got)
	}
	if took > time.Second {
		t.Errorf("took %s, want about the 50ms timeout", took)
	}
	if !strings.Contains(logs, "source=cptm") {
		t.Errorf("logs = %q, want the timeout logged for cptm", logs)
	}
}

// R4.2
func TestCollectAbandonsASourceThatIgnoresTheContext(t *testing.T) {
	stuck := &namedFake{name: "metro", block: make(chan struct{})}
	t.Cleanup(func() { close(stuck.block) })

	snaps, _, took := collect(t, 50*time.Millisecond, ok("proximotrem", 1), stuck)

	if got := names(snaps); got != "proximotrem" || took > time.Second {
		t.Errorf("sources = %s after %s, want proximotrem after about 50ms", got, took)
	}
}

// R4.2 with two stuck sources: each one is abandoned at the deadline, not
// only the first.
func TestCollectAbandonsEverySourceThatIgnoresTheContext(t *testing.T) {
	stuckA := &namedFake{name: "metro", block: make(chan struct{})}
	stuckB := &namedFake{name: "cptm", block: make(chan struct{})}
	t.Cleanup(func() { close(stuckA.block); close(stuckB.block) })

	done := make(chan []Snapshot, 1)
	go func() {
		snaps, _, _ := collect(t, 50*time.Millisecond, ok("proximotrem", 1), stuckA, stuckB)
		done <- snaps
	}()
	select {
	case snaps := <-done:
		if got := names(snaps); got != "proximotrem" {
			t.Errorf("sources = %s, want proximotrem", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Collect still waiting 2 s after a 50 ms timeout")
	}
}

// A reading ready right at the deadline is kept, never dropped as a timeout.
func TestCollectKeepsAnAnswerReadyAtTheDeadline(t *testing.T) {
	for range 50 {
		snaps, _, _ := collect(t, time.Millisecond, ok("proximotrem", 1))
		if len(snaps) != 1 {
			t.Fatalf("sources = %s, want the answer that was already there", names(snaps))
		}
	}
}

// R18 / R5: a source answering with no line at all counts as failed, so a
// cycle where every source is empty is a failed cycle, not an empty board.
func TestCollectTreatsAnEmptyAnswerAsAFailure(t *testing.T) {
	empty := &namedFake{name: "cptm-novo"}

	snaps, logs, _ := collect(t, time.Second, ok("proximotrem", 1), empty)

	if got := names(snaps); got != "proximotrem" {
		t.Errorf("sources = %s, want proximotrem", got)
	}
	if !strings.Contains(logs, "source=cptm-novo") {
		t.Errorf("logs = %q, want the empty source logged", logs)
	}
}

// R4.3
func TestCollectRunsTheSourcesInParallel(t *testing.T) {
	var sources []StatusSource
	for _, n := range []string{"proximotrem", "metro", "cptm-novo", "cptm"} {
		f := ok(n, 1)
		f.delay = 100 * time.Millisecond
		sources = append(sources, f)
	}

	snaps, _, took := collect(t, time.Second, sources...)

	if len(snaps) != 4 || took >= 300*time.Millisecond {
		t.Errorf("%d sources in %s, want 4 in about 100ms (not the 400ms sum)", len(snaps), took)
	}
}

// R13.1 and R22.1: every failure is logged under its own source name.
func TestCollectLogsEveryFailureUnderItsOwnName(t *testing.T) {
	cptm := &namedFake{name: "cptm", err: errors.New("legacy down")}
	cptmNovo := &namedFake{name: "cptm-novo", err: errors.New("site down")}

	_, logs, _ := collect(t, time.Second, ok("proximotrem", 1), cptm, cptmNovo)

	for _, want := range []string{"source=cptm ", "legacy down", "source=cptm-novo", "site down"} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs = %q, want %q", logs, want)
		}
	}
}

// R13.2
func TestCollectHealthySourcesLogNoFailure(t *testing.T) {
	_, logs, _ := collect(t, time.Second, ok("proximotrem", 1), ok("cptm", 1))

	if logs != "" {
		t.Errorf("logs = %q, want none", logs)
	}
}

// R14.1: the poller is now the one reading sources, so it logs new labels.
func TestCollectLogsUnrecognizedLabels(t *testing.T) {
	odd := &namedFake{name: "cptm", snap: Snapshot{Readings: []Reading{
		{Code: 8, State: status.Desconhecido, Recognized: false, Label: "Operação Especial"},
	}}}

	_, logs, _ := collect(t, time.Second, odd)

	for _, want := range []string{"unrecognized status label", "source=cptm", "line=8", `label="Operação Especial"`} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs = %q, want %q", logs, want)
		}
	}
}
