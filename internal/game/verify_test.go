package game_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/provider"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// verifyTestStream / verifyTestProvider mirror the inspector's
// scripted provider but live in package game's _test files.
type verifyTestStream struct {
	chunks []provider.StreamChunk
	i      int
}

func (s *verifyTestStream) Next(_ context.Context) (provider.StreamChunk, error) {
	if s.i >= len(s.chunks) {
		return provider.StreamChunk{}, io.EOF
	}
	c := s.chunks[s.i]
	s.i++
	return c, nil
}
func (s *verifyTestStream) Close() error { return nil }

type verifyTestProvider struct{}

func (p *verifyTestProvider) Info() provider.Info {
	return provider.Info{ID: "scripted", APIVersion: "v0"}
}

func (p *verifyTestProvider) Stream(_ context.Context, _ *provider.Request) (provider.EventStream, error) {
	usage := provider.UsageUpdate{InputTokens: 5, OutputTokens: 5}
	return &verifyTestStream{
		chunks: []provider.StreamChunk{
			{Kind: provider.ChunkText, Text: "ok."},
			{Kind: provider.ChunkUsage, Usage: &usage},
			{Kind: provider.ChunkEnd, StopReason: "stop"},
		},
	}, nil
}

func writeRealLog(t *testing.T, n int) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "case.db")
	log, err := eventlog.NewSQLite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	d := witness.NewLiveDriverWith(&verifyTestProvider{}, "test-model", nil, log, streetlight.Case.Beliefs)
	for i := 0; i < n; i++ {
		if _, err := d.Respond(context.Background(), "the bag", kase.Directly, nil); err != nil {
			t.Fatalf("respond %d: %v", i, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestVerify_OKOnIntactLog(t *testing.T) {
	path := writeRealLog(t, 2)
	r, err := game.Verify(context.Background(), path)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !r.OK {
		t.Errorf("Verify reported broken: %s", r.Reason)
	}
	if r.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2", r.RunCount)
	}
	if r.EventCount == 0 {
		t.Errorf("EventCount = 0")
	}
	if len(r.FirstHash) == 0 || len(r.LastHash) == 0 {
		t.Errorf("hashes empty")
	}
}

func TestVerify_DetectsTampering(t *testing.T) {
	path := writeRealLog(t, 1)
	// Corrupt by appending random bytes to the .db file. Any change
	// to the body breaks SQLite's page integrity, so the read itself
	// fails — that's still a verify failure, just at the open stage.
	// To get a clean "chain broken" reason, we can instead overwrite
	// a payload byte after opening; for now, append-corrupt is
	// sufficient evidence verify catches mutation.
	f, err := os.OpenFile(path, os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for tamper: %v", err)
	}
	if _, err := f.Seek(100, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{0xff, 0xff, 0xff, 0xff}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	r, err := game.Verify(context.Background(), path)
	// Either Verify returns an error opening the corrupted DB, or
	// it succeeds opening but reports OK=false. Both prove tamper
	// detection.
	if err == nil && r.OK {
		t.Errorf("tampered log verified as OK")
	}
}

func TestVerify_NoFile(t *testing.T) {
	if _, err := game.Verify(context.Background(), ""); err == nil {
		t.Error("expected error for empty path")
	}
}
