package witness_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/provider"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// scriptedStream replays a fixed slice of StreamChunks. Each Next()
// returns the next chunk; Next after the last chunk returns io.EOF.
type scriptedStream struct {
	chunks []provider.StreamChunk
	i      int
}

func (s *scriptedStream) Next(_ context.Context) (provider.StreamChunk, error) {
	if s.i >= len(s.chunks) {
		return provider.StreamChunk{}, io.EOF
	}
	c := s.chunks[s.i]
	s.i++
	return c, nil
}

func (s *scriptedStream) Close() error { return nil }

// scriptedProvider returns the same scripted response for every Stream
// call. Sufficient for a smoke test where each ask gets a canned line
// without any tool use roundtrip.
type scriptedProvider struct {
	info provider.Info
	text string
}

func (p *scriptedProvider) Info() provider.Info { return p.info }

func (p *scriptedProvider) Stream(_ context.Context, _ *provider.Request) (provider.EventStream, error) {
	usage := provider.UsageUpdate{InputTokens: 50, OutputTokens: 10}
	return &scriptedStream{
		chunks: []provider.StreamChunk{
			{Kind: provider.ChunkText, Text: p.text},
			{Kind: provider.ChunkUsage, Usage: &usage},
			{Kind: provider.ChunkEnd, StopReason: "stop"},
		},
	}, nil
}

// TestLiveDriver_RoundTripSQLite runs two asks against a scripted
// provider, writing to a per-test SQLite event log, and confirms each
// ask produced its own RunCompleted'd run with the canned text.
func TestLiveDriver_RoundTripSQLite(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "case-sess.db")

	prov := &scriptedProvider{
		info: provider.Info{ID: "scripted", APIVersion: "v0"},
		text: "she pauses.",
	}

	log, err := eventlog.NewSQLite(path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	d := witness.NewLiveDriverWith(prov, "test-model", nil, log)
	defer d.Close()

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		got, err := d.Respond(ctx, "the bag", kase.Directly, nil)
		if err != nil {
			t.Fatalf("Respond %d: %v", i, err)
		}
		if got.Text != "she pauses." {
			t.Errorf("Respond %d text = %q, want %q", i, got.Text, "she pauses.")
		}
		if got.OutputTokens != 10 {
			t.Errorf("Respond %d OutputTokens = %d, want 10 (scripted usage)", i, got.OutputTokens)
		}
	}

	// Re-read the log read-only and confirm there are two distinct,
	// terminated runs.
	roLog, err := eventlog.NewSQLite(path, eventlog.WithReadOnly())
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer roLog.Close()
	lister, ok := roLog.(eventlog.RunLister)
	if !ok {
		t.Fatal("sqlite log does not implement RunLister")
	}
	runs, err := lister.ListRuns(ctx)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("run count = %d, want 2", len(runs))
	}
	for _, r := range runs {
		if r.TerminalKind != event.KindRunCompleted {
			t.Errorf("run %s terminal = %s, want RunCompleted", r.RunID, r.TerminalKind)
		}
		evs, err := roLog.Read(ctx, r.RunID)
		if err != nil {
			t.Fatalf("Read %s: %v", r.RunID, err)
		}
		if err := eventlog.Validate(evs); err != nil {
			t.Errorf("validate %s: %v", r.RunID, err)
		}
	}
}

// TestNewLiveProviderFromEnv_NoKeys returns a clear error when the
// environment is empty so the caller knows to fall back to the stub.
func TestNewLiveProviderFromEnv_NoKeys(t *testing.T) {
	t.Setenv("PROVIDER", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	_, err := witness.NewLiveProviderFromEnv()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, errors.New("")) {
		t.Errorf("error message empty: %v", err)
	}
}
