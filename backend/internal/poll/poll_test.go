package poll

import (
	"cmp"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
	"github.com/DouglasDans/status-trens-sp/backend/internal/notify"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// A line the catalog does not know, so presentation comes from the source.
const code = 9999

type fakeSource struct {
	name string
	snap source.Snapshot
	err  error
}

func (f *fakeSource) Name() string { return cmp.Or(f.name, "fake") }

func (f *fakeSource) Fetch(context.Context) (source.Snapshot, error) {
	snap := f.snap
	snap.Source = f.Name()
	return snap, f.err
}

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

type fakeBoards struct {
	board   board.Board
	found   bool
	saves   int
	saveErr error
	loadErr error
}

func (f *fakeBoards) SaveBoard(_ context.Context, b board.Board) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saves++
	f.board, f.found = b, true
	return nil
}

func (f *fakeBoards) LoadBoard(context.Context) (board.Board, bool, error) {
	return f.board, f.found, f.loadErr
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

var t0 = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func setup() (*Poller, *fakeSource, *fakeStore, *fakeNotifier) {
	p, srcs, st, n, _, _ := setupMany(1)
	return p, srcs[0], st, n
}

// setupMany wires n sources, in priority order.
func setupMany(count int) (*Poller, []*fakeSource, *fakeStore, *fakeNotifier, *fakeBoards, *fakeClock) {
	st, n, b, clock := &fakeStore{lines: map[int]status.Memory{}}, &fakeNotifier{}, &fakeBoards{}, &fakeClock{t0}
	var srcs []*fakeSource
	var named []source.StatusSource
	for i := range count {
		s := &fakeSource{name: []string{"proximotrem", "metro", "cptm-novo", "cptm"}[i]}
		srcs = append(srcs, s)
		named = append(named, s)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := &Poller{Sources: named, Timeout: time.Second, Store: st, Boards: b, Notifier: n, Logger: logger, Now: clock.now}
	return p, srcs, st, n, b, clock
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

func (f *fakeSource) line(c int, s status.State, label, desc string) *fakeSource {
	f.err = nil
	f.snap.Readings = []source.Reading{{Code: c, State: s, Recognized: true, Label: label, Description: desc}}
	return f
}

func (f *fakeSource) fail() { f.err = errors.New(f.Name() + " down") }

// R5.2: one failed source does not abort the cycle.
func TestOneSourceFailureDoesNotAbortTheCycle(t *testing.T) {
	p, srcs, st, n, b, _ := setupMany(4)
	ctx := context.Background()
	for _, s := range srcs {
		s.line(3, status.Normal, "Operação Normal", "")
	}
	_ = p.Run(ctx)
	srcs[0].line(3, status.Paralisada, "Paralisada", "")
	srcs[3].fail()

	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(n.sent) != 1 || st.lines[3].State != status.Paralisada {
		t.Errorf("sent %v, memory %+v; want the change pushed", n.sent, st.lines[3])
	}
	if got := b.board.Sources; len(got) != 3 {
		t.Errorf("board sources = %v, want the 3 that answered", got)
	}
}

// R11.1: the push carries the text of the source that decided the state.
func TestPushCarriesTheWinningSourceText(t *testing.T) {
	p, srcs, _, n, _, _ := setupMany(2)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "Linha opera normalmente.")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	srcs[1].line(3, status.Paralisada, "Paralisada", "Falha de energia entre Luz e Brás.")
	_ = p.Run(ctx)

	if len(n.sent) != 1 || n.sent[0].Body != "Paralisada · Falha de energia entre Luz e Brás." {
		t.Errorf("sent %+v, want the metro text", n.sent)
	}
}

// R12.1: two sources reporting the same change push once.
func TestOnePushPerLinePerCycle(t *testing.T) {
	p, srcs, _, n, _, _ := setupMany(3)
	ctx := context.Background()
	for _, s := range srcs {
		s.line(3, status.Normal, "Operação Normal", "")
	}
	_ = p.Run(ctx)
	srcs[1].line(3, status.Paralisada, "Paralisada", "")
	srcs[2].line(3, status.Paralisada, "Paralisada", "")

	_ = p.Run(ctx)

	if len(n.sent) != 1 {
		t.Errorf("sent %d pushes, want 1", len(n.sent))
	}
}

// R7.2: the line stays stopped while one source still reports it.
func TestStaysStoppedWhileOneSourceStillReportsIt(t *testing.T) {
	p, srcs, _, n, _, _ := setupMany(2)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	srcs[0].line(3, status.Paralisada, "Paralisada", "")
	srcs[1].line(3, status.Paralisada, "Paralisada", "")
	_ = p.Run(ctx)

	srcs[0].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	if len(n.sent) != 1 {
		t.Errorf("sent %+v, want only the stop (no early all clear)", n.sent)
	}
}

// R12.2: documents the accepted risk: when the only source reporting a
// problem fails, the line goes back to normal and notifies, then again.
func TestFlappingSourceNotifiesEveryChange(t *testing.T) {
	p, srcs, _, n, _, _ := setupMany(2)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	srcs[1].line(3, status.Paralisada, "Paralisada", "")
	_ = p.Run(ctx)
	srcs[1].fail()
	_ = p.Run(ctx)
	srcs[1].line(3, status.Paralisada, "Paralisada", "")
	_ = p.Run(ctx)

	if len(n.sent) != 3 {
		t.Errorf("sent %d pushes, want 3 (stop, normal, stop)", len(n.sent))
	}
}

// R12.3: another source winning within the same state pushes nothing.
func TestWinnerChangeWithinStateDoesNotNotify(t *testing.T) {
	p, srcs, _, n, b, _ := setupMany(2)
	ctx := context.Background()
	srcs[0].line(3, status.Reduzida, "Atividade Programada", "")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)

	srcs[0].line(3, status.Normal, "Operação Normal", "")
	srcs[1].line(3, status.Reduzida, "Velocidade Reduzida", "")
	_ = p.Run(ctx)

	if len(n.sent) != 0 {
		t.Errorf("sent %+v, want nothing", n.sent)
	}
	if got := b.board.Lines[0].Label; got != "Velocidade Reduzida" {
		t.Errorf("board label = %q, want the new winner's", got)
	}
}

// R15.1: the board is saved every cycle, even without changes.
func TestBoardIsSavedEveryCycle(t *testing.T) {
	p, srcs, _, _, b, clock := setupMany(1)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	clock.t = t0.Add(time.Minute)

	_ = p.Run(ctx)

	if b.saves != 2 || !b.board.UpdatedAt.Equal(t0.Add(time.Minute)) {
		t.Errorf("saves = %d, updatedAt = %s; want 2 saves, the second cycle's time", b.saves, b.board.UpdatedAt)
	}
}

// R25.1-R25.3: a line only one source covers keeps its last reading for
// 10 min, then shows unknown; its return only pushes a real change.
func TestLineWithoutVotesKeepsItsLastReadingThenUnknown(t *testing.T) {
	p, srcs, _, n, b, clock := setupMany(2)
	ctx := context.Background()
	srcs[0].line(6, status.Normal, "Operação Normal", "")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	srcs[0].line(6, status.Paralisada, "Paralisada", "")
	_ = p.Run(ctx) // push 1: line 6 stopped

	srcs[0].fail()
	clock.t = t0.Add(9 * time.Minute)
	_ = p.Run(ctx)
	if got := line(b.board, 6); got.State != status.Paralisada {
		t.Errorf("after 9 min line 6 = %+v, want its last reading", got)
	}

	clock.t = t0.Add(15 * time.Minute)
	_ = p.Run(ctx)
	if got := line(b.board, 6); got.State != status.Desconhecido {
		t.Errorf("after 15 min line 6 = %+v, want unknown", got)
	}
	if len(n.sent) != 1 {
		t.Errorf("sent %d pushes while unseen, want only the stop", len(n.sent))
	}

	srcs[0].line(6, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	if len(n.sent) != 2 || n.sent[1].Line != 6 {
		t.Errorf("sent %+v, want the all clear for line 6 on its return", n.sent)
	}
}

// R18.1: every source failing keeps the last board.
func TestAllSourcesFailingKeepsTheLastBoard(t *testing.T) {
	p, srcs, _, n, b, clock := setupMany(2)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "")
	srcs[1].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	srcs[0].fail()
	srcs[1].fail()
	clock.t = t0.Add(time.Minute)

	if err := p.Run(ctx); err == nil {
		t.Error("Run() = nil, want an error")
	}
	if b.saves != 1 || !b.board.UpdatedAt.Equal(t0) || len(n.sent) != 0 {
		t.Errorf("saves = %d, updatedAt = %s, sent = %v; want the first board untouched", b.saves, b.board.UpdatedAt, n.sent)
	}
}

// R15.2: a failed board save is reported, but the pushes still go out.
func TestBoardSaveFailureStillPushes(t *testing.T) {
	p, srcs, _, n, b, _ := setupMany(1)
	ctx := context.Background()
	srcs[0].line(3, status.Normal, "Operação Normal", "")
	_ = p.Run(ctx)
	srcs[0].line(3, status.Paralisada, "Paralisada", "")
	b.saveErr = errors.New("throttled")

	if err := p.Run(ctx); err == nil {
		t.Error("Run() = nil, want the save error")
	}
	if len(n.sent) != 1 {
		t.Errorf("sent %d pushes, want the change pushed anyway", len(n.sent))
	}
}

// R15.5: without the line memory the board is still saved, but nothing is
// pushed.
func TestBoardSavedWhenMemoryLoadFails(t *testing.T) {
	p, srcs, st, n, b, _ := setupMany(1)
	srcs[0].line(3, status.Paralisada, "Paralisada", "")
	st.err = errors.New("store down")

	if err := p.Run(context.Background()); err == nil {
		t.Error("Run() = nil, want the memory error")
	}
	if b.saves != 1 || len(n.sent) != 0 {
		t.Errorf("board saves = %d, sent = %v; want the board saved and nothing pushed", b.saves, n.sent)
	}
}

func line(b board.Board, code int) board.Line {
	for _, l := range b.Lines {
		if l.Code == code {
			return l
		}
	}
	return board.Line{}
}
