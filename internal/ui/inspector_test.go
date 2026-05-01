package ui

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/provider"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// inspectorTestStream replays a small chunk slice for one ask. Kept
// separate from witness/live_test.go's scriptedProvider to avoid
// cross-package test imports.
type inspectorTestStream struct {
	chunks []provider.StreamChunk
	i      int
}

func (s *inspectorTestStream) Next(_ context.Context) (provider.StreamChunk, error) {
	if s.i >= len(s.chunks) {
		return provider.StreamChunk{}, io.EOF
	}
	c := s.chunks[s.i]
	s.i++
	return c, nil
}

func (s *inspectorTestStream) Close() error { return nil }

type inspectorTestProvider struct{}

func (p *inspectorTestProvider) Info() provider.Info {
	return provider.Info{ID: "scripted", APIVersion: "v0"}
}

func (p *inspectorTestProvider) Stream(_ context.Context, _ *provider.Request) (provider.EventStream, error) {
	usage := provider.UsageUpdate{InputTokens: 10, OutputTokens: 5}
	return &inspectorTestStream{
		chunks: []provider.StreamChunk{
			{Kind: provider.ChunkText, Text: "ok."},
			{Kind: provider.ChunkUsage, Usage: &usage},
			{Kind: provider.ChunkEnd, StopReason: "stop"},
		},
	}, nil
}

func TestInspector_LoadsRealEvents(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "case.db")
	log, err := eventlog.NewSQLite(path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	d := witness.NewLiveDriverWith(&inspectorTestProvider{}, "test-model", nil, log, streetlight.Case.Beliefs)
	defer d.Close()
	for i := 0; i < 2; i++ {
		if _, err := d.Respond(context.Background(), "the bag", kase.Directly, nil); err != nil {
			t.Fatalf("respond %d: %v", i, err)
		}
	}

	m := newInspector(path)
	if m.loadErr != "" {
		t.Fatalf("loadErr = %q, want empty", m.loadErr)
	}
	if len(m.events) == 0 {
		t.Errorf("inspector loaded 0 events; expected at least 2 runs' worth")
	}
}

func TestInspector_EmptyPathReportsError(t *testing.T) {
	m := newInspector("")
	if m.loadErr == "" {
		t.Errorf("expected loadErr for empty path")
	}
	// View should still render without panicking.
	_ = m.View()
}
