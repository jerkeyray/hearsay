package game_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// TestSession_ConcurrentAskAndRead simulates the Bubble Tea pattern:
// Update dispatches Ask in a goroutine while View reads Log /
// ClockDisplay on the main loop. Run with `go test -race` to verify
// the mutex correctness.
func TestSession_ConcurrentAskAndRead(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{
		ID: "test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true},
		},
	}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 100_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	const asks = 50
	const readers = 4

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader goroutines: simulate View() calls.
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = s.Log()
					_ = s.ClockDisplay()
					_ = s.RemainingOutputTokens()
					_ = s.SessionEnded()
				}
			}
		}()
	}

	// Single writer: Ask in sequence (matches the Bubble Tea pattern
	// where only one ask is in flight at a time).
	for i := 0; i < asks; i++ {
		if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	close(stop)
	wg.Wait()

	if got := s.TurnCount(); got != asks {
		t.Errorf("TurnCount = %d, want %d", got, asks)
	}
}
