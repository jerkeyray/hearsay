package ui

import (
	"context"
	"crypto/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oklog/ulid/v2"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// DriverFactory returns a fresh witness.Driver for one session. The
// caller may build either a live or stub driver. savePath is the
// SQLite path the live driver should write into; stub drivers ignore
// it.
type DriverFactory func(ctx context.Context, c kase.Case, savePath string) (witness.Driver, error)

type screen int

const (
	screenSplash screen = iota
	screenBriefing
	screenInterrogation
	screenError
	screenPlaceholder
)

type model struct {
	screen        screen
	splash        splashModel
	briefing      briefingModel
	interrogation interrogationModel
	makeDriver    DriverFactory
	errMsg        string
	placeholder   string
	quitting      bool
}

// New constructs the root TUI model. makeDriver builds a fresh
// Driver per session; when nil, the stub driver is used.
func New(makeDriver DriverFactory) tea.Model {
	if makeDriver == nil {
		makeDriver = func(_ context.Context, _ kase.Case, _ string) (witness.Driver, error) {
			return witness.NewStubDriver(), nil
		}
	}
	return model{
		screen:     screenSplash,
		splash:     newSplash(),
		makeDriver: makeDriver,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenSplash:
		next, cmd, choice, picked := m.splash.Update(msg)
		m.splash = next
		if !picked {
			return m, cmd
		}
		switch choice {
		case choiceQuit:
			m.quitting = true
			return m, tea.Quit
		case choiceNew:
			m.briefing = newBriefing(streetlight.Case)
			m.screen = screenBriefing
		case choiceContinue:
			m.screen = screenPlaceholder
			m.placeholder = "continue — save picker lands here later"
		}
		return m, nil

	case screenBriefing:
		next, cmd, advance, back := m.briefing.Update(msg)
		m.briefing = next
		if back {
			m.screen = screenSplash
			return m, nil
		}
		if advance {
			ctx := context.Background()
			savePath, err := newSavePath(streetlight.Case.ID)
			if err != nil {
				m.errMsg = err.Error()
				m.screen = screenError
				return m, nil
			}
			driver, err := m.makeDriver(ctx, streetlight.Case, savePath)
			if err != nil {
				m.errMsg = err.Error()
				m.screen = screenError
				return m, nil
			}
			session, err := game.NewSession(ctx, streetlight.Case, driver)
			if err != nil {
				_ = driver.Close()
				m.errMsg = err.Error()
				m.screen = screenError
				return m, nil
			}
			m.interrogation = newInterrogation(session)
			m.screen = screenInterrogation
			return m, nil
		}
		return m, cmd

	case screenInterrogation:
		next, cmd, back := m.interrogation.Update(msg)
		m.interrogation = next
		if back {
			_ = m.interrogation.Close(context.Background())
			m.screen = screenSplash
			return m, nil
		}
		return m, cmd

	case screenError:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc", "enter":
				m.errMsg = ""
				m.screen = screenSplash
			}
		}
		return m, nil

	case screenPlaceholder:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "q", "ctrl+c", "esc":
				m.screen = screenSplash
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenSplash:
		return m.splash.View()
	case screenBriefing:
		return m.briefing.View()
	case screenInterrogation:
		return m.interrogation.View()
	case screenError:
		return styleBorder.Render(
			styleTitle.Render("hearsay") + "\n\n" +
				"could not start session:\n  " + m.errMsg + "\n\n" +
				styleDim.Render("esc back · q quit"))
	case screenPlaceholder:
		return styleBorder.Render(m.placeholder + "\n\n" + styleDim.Render("esc back · q quit"))
	}
	return ""
}

// newSavePath returns a fresh SQLite save path under the user's
// hearsay home directory: <saveDir>/<caseID>-<sessionID>.db.
func newSavePath(caseID string) (string, error) {
	saveDir, err := game.EnsureSaveDir()
	if err != nil {
		return "", err
	}
	sessionID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
	return game.SavePath(saveDir, caseID, sessionID), nil
}
