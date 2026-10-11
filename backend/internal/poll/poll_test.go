package poll

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/notify"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// A line the catalog does not know, so presentation comes from the source.
const code = 9999

type fakeSource struct {
	snap source.Snapshot
	err  error
}

func (f *fakeSource) Fetch(context.Context) (source.Snapshot, error) { return f.snap, f.err }

func (f *fakeSource) report(s status.State, label, desc string) {
	f.snap = source.Snapshot{Source: "fake", Readings: []source.Reading{
		{Code: code, Name: "Teste", ColorHex: "#123456", State: s, Label: label, Description: desc},
	}}
}

type fakeStore struct {
	lines map[int]status.Memory
	saves int
	err   error
}

func (f *fakeStore) Load(context.Context) (map[int]status.Memory, error) {
	out := map[int]status.Memory{}
	for k, v := range f.lines {
		out[k] = v
	}
	return out, f.err
}

func (f *fakeStore) Save(_ context.Context, code int, m status.Memory) error {
	f.saves++
	f.lines[code] = m
	return nil
}

type fakeNotifier struct {
	sent []notify.Message
	err  error
}

func (f *fakeNotifier) Send(_ context.Context, m notify.Message) error {
	f.sent = append(f.sent, m)
	return f.err
}

func setup() (*Poller, *fakeSource, *fakeStore, *fakeNotifier) {
	src, st, n := &fakeSource{}, &fakeStore{lines: map[int]status.Memory{}}, &fakeNotifier{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Poller{Source: src, Store: st, Notifier: n, Logger: logger}, src, st, n
}

func TestFirstPollOnlyRecordsABaseline(t *testing.T) {
	p, src, st, n := setup()
	src.report(status.Paralisada, "Paralisada", "")

	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(n.sent) != 0 {
		t.Errorf("sent %v on the baseline", n.sent)
	}
	if st.lines[code].State != status.Paralisada {
		t.Errorf("memory = %+v, want the baseline saved", st.lines[code])
	}
}

func TestAChangeIsPushedWithTheOperatorText(t *testing.T) {
	p, src, _, n := setup()
	ctx := context.Background()
	src.report(status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	src.report(status.Reduzida, "Velocidade Reduzida", "Falha em equipamento.")
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	want := notify.Message{Line: code, Title: "Linha 9999 – Teste", Body: "Velocidade Reduzida · Falha em equipamento.", ColorHex: "#123456"}
	if len(n.sent) != 1 || n.sent[0] != want {
		t.Errorf("sent %+v, want [%+v]", n.sent, want)
	}
}

func TestAFailedPushIsRetriedOnTheNextPoll(t *testing.T) {
	p, src, _, n := setup()
	ctx := context.Background()
	src.report(status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	src.report(status.Paralisada, "Paralisada", "")

	n.err = errors.New("fcm down")
	_ = p.Run(ctx)
	n.err = nil
	_ = p.Run(ctx)

	if len(n.sent) != 2 {
		t.Errorf("sent %d times, want the failed push tried again", len(n.sent))
	}
	_ = p.Run(ctx)
	if len(n.sent) != 2 {
		t.Errorf("sent %d times, want no repeat after a successful push", len(n.sent))
	}
}

func TestNothingIsWrittenWhenNothingChanged(t *testing.T) {
	p, src, st, _ := setup()
	ctx := context.Background()
	src.report(status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	_ = p.Run(ctx)

	if st.saves != 1 {
		t.Errorf("saves = %d, want only the baseline", st.saves)
	}
}

func TestSourceFailureChangesNothing(t *testing.T) {
	p, src, st, n := setup()
	src.err = errors.New("all sources down")

	if err := p.Run(context.Background()); err == nil {
		t.Error("Run() = nil, want the source error")
	}
	if st.saves != 0 || len(n.sent) != 0 {
		t.Errorf("saves = %d, sent = %v; want nothing", st.saves, n.sent)
	}
}

func TestStoreFailureStopsThePoll(t *testing.T) {
	p, src, st, n := setup()
	src.report(status.Normal, "Operação Normal", "")
	st.err = errors.New("store down")

	if err := p.Run(context.Background()); err == nil {
		t.Error("Run() = nil, want the store error")
	}
	if len(n.sent) != 0 {
		t.Errorf("sent %v without memory", n.sent)
	}
}
