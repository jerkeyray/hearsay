// Package game owns the engine-side session state: the current case,
// the conversation log, the per-session SQLite event journal, and the
// turn-coordination seam where Starling event-log writes attach and a
// real LLM driver replaces the stub (M2). The UI holds cursor state
// and renders; everything that would survive a process restart lives
// here.
package game

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// Exchange is one turn in the conversation: what the player asked and
// what the witness said back.
type Exchange struct {
	Turn      int
	TurnID    string
	Topic     string
	Technique kase.Technique
	Witness   string
}

// Session is the per-case engine state. One Session per save file.
type Session struct {
	Case     kase.Case
	RunID    string
	savePath string
	witness  *witness.Agent
	journal  *Journal
	log      []Exchange
}

// NewSession opens a journal for a fresh run, writes RunStarted, and
// returns a ready-to-Ask session. Caller must Close when done.
func NewSession(ctx context.Context, c kase.Case, w *witness.Agent) (*Session, error) {
	saveDir, err := EnsureSaveDir()
	if err != nil {
		return nil, fmt.Errorf("session: ensure save dir: %w", err)
	}
	runID := newULID()
	path := SavePath(saveDir, c.ID, runID)
	j, err := OpenJournal(ctx, path, runID)
	if err != nil {
		return nil, fmt.Errorf("session: open journal: %w", err)
	}
	if err := j.AppendRunStarted(ctx, c.ID); err != nil {
		j.Close()
		return nil, fmt.Errorf("session: write run started: %w", err)
	}
	return &Session{
		Case:     c,
		RunID:    runID,
		savePath: path,
		witness:  w,
		journal:  j,
	}, nil
}

// Ask runs one turn: write TurnStarted, ask the witness, write
// AssistantMessageCompleted with the witness line. The exchange is
// appended to the in-memory log and returned for rendering.
func (s *Session) Ask(ctx context.Context, topic string, technique kase.Technique) (Exchange, error) {
	turnID := newULID()
	if err := s.journal.AppendTurnStarted(ctx, turnID); err != nil {
		return Exchange{}, err
	}
	line := s.witness.Respond(topic, technique)
	if err := s.journal.AppendAssistantMessage(ctx, turnID, line); err != nil {
		return Exchange{}, err
	}
	ex := Exchange{
		Turn:      len(s.log),
		TurnID:    turnID,
		Topic:     topic,
		Technique: technique,
		Witness:   line,
	}
	s.log = append(s.log, ex)
	return ex, nil
}

// Close releases the journal. M1 does not write a terminal event; see
// Journal.Close for context. Safe to call multiple times.
func (s *Session) Close(ctx context.Context) error {
	if s.journal == nil {
		return nil
	}
	err := s.journal.Close()
	s.journal = nil
	return err
}

// Log returns the exchanges in order. Read-only view for renderers.
func (s *Session) Log() []Exchange { return s.log }

// TurnCount is the number of completed turns.
func (s *Session) TurnCount() int { return len(s.log) }

// SavePath returns the SQLite path this session writes to. Remains
// valid after Close.
func (s *Session) SavePath() string { return s.savePath }

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
