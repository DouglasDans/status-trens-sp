package store

import (
	"context"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

func TestInMemoryStartsEmpty(t *testing.T) {
	lines, err := NewInMemory().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("Load() = %v, want empty", lines)
	}
}

func TestInMemoryKeepsWhatWasSaved(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory()
	want := status.Memory{State: status.Paralisada, LastNotified: status.Paralisada}

	if err := s.Save(ctx, 9, want); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, 7, status.Memory{State: status.Normal}); err != nil {
		t.Fatal(err)
	}

	lines, err := s.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lines[9] != want || len(lines) != 2 {
		t.Errorf("Load() = %v, want line 9 = %v among 2 lines", lines, want)
	}
}

func TestInMemoryLoadReturnsACopy(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory()
	_ = s.Save(ctx, 9, status.Memory{State: status.Normal})

	lines, _ := s.Load(ctx)
	lines[9] = status.Memory{State: status.Paralisada}

	again, _ := s.Load(ctx)
	if again[9].State != status.Normal {
		t.Errorf("store changed through a loaded map: %v", again[9])
	}
}
