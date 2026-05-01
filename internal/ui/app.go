package ui

import (
	"context"
	"crypto/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oklog/ulid/v2"

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
	screenCasePicker
	screenBriefing
	screenInterrogation
	screenReconstruction
	screenVerdict
	screenError
	screenPlaceholder
)

type model struct {
	screen         screen
	splash         splashModel
	casePicker     casePickerModel
	briefing       briefingModel
	interrogation  interrogationModel
	reconstruction reconstructionModel
	verdict        verdictModel
	help           helpModel
	helpOpen       bool
	cases          []kase.Case
	session        *game.Session // active session, nil between cases
	makeDriver     DriverFactory
	errMsg         string
	placeholder    string
	quitting       bool
}

// New constructs the root TUI model. makeDriver builds a fresh
// Driver per session; when nil, the stub driver is used. cases is
// the list of available cases; an empty list disables "new case".
func New(makeDriver DriverFactory, cases []kase.Case) tea.Model {
	if makeDriver == nil {
		makeDriver = func(_ context.Context, _ kase.Case, _ string) (witness.Driver, error) {
			return witness.NewStubDriver(), nil
		}
	}
	return model{
		screen:     screenSplash,
		splash:     newSplash(),
		cases:      cases,
		makeDriver: makeDriver,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Help overlay: captures all keys while open.
	if m.helpOpen {
		next, cmd, done := m.help.Update(msg)
		m.help = next
		if done {
			m.helpOpen = false
		}
		return m, cmd
	}
	// "?" opens help globally — but not on the reconstruction screen
	// where it's a printable character users may want to type.
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "?" && m.screen != screenReconstruction {
		m.help = newHelp()
		m.helpOpen = true
		return m, nil
	}

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
			if len(m.cases) == 0 {
				m.errMsg = "no cases compiled in"
				m.screen = screenError
				return m, nil
			}
			if len(m.cases) == 1 {
				m = m.openCase(m.cases[0])
				return m, nil
			}
			m.casePicker = newCasePicker(m.cases)
			m.screen = screenCasePicker
		case choiceContinue:
			m.screen = screenPlaceholder
			m.placeholder = "continue — save picker lands here later"
		}
		return m, nil

	case screenCasePicker:
		next, cmd, picked, back := m.casePicker.Update(msg)
		m.casePicker = next
		if back {
			m.screen = screenSplash
			return m, nil
		}
		if picked {
			c := m.casePicker.Selected()
			m.briefing = newBriefing(c)
			m.screen = screenBriefing
			return m, nil
		}
		return m, cmd

	case screenBriefing:
		next, cmd, advance, back := m.briefing.Update(msg)
		m.briefing = next
		if back {
			if len(m.cases) > 1 {
				m.screen = screenCasePicker
			} else {
				m.screen = screenSplash
			}
			return m, nil
		}
		if advance {
			m = m.openCase(m.briefing.kase)
			return m, nil
		}
		return m, cmd

	case screenInterrogation:
		// Branch completion: swap the active session.
		if b, ok := msg.(branchedMsg); ok {
			if b.err != nil {
				m.errMsg = b.err.Error()
				m.screen = screenError
				return m, nil
			}
			// Close the parent session's driver — the child owns
			// its own copy now. The parent's SQLite history is on
			// disk, untouched.
			_ = m.session.Close(context.Background())
			m.session = b.session
			m.interrogation = newInterrogation(b.session)
			return m, nil
		}
		next, cmd, back, done := m.interrogation.Update(msg)
		m.interrogation = next
		if done {
			m.reconstruction = newReconstruction(m.session.Case.Reconstruction)
			m.screen = screenReconstruction
			return m, nil
		}
		if back {
			_ = m.interrogation.Close(context.Background())
			m.session = nil
			m.screen = screenSplash
			return m, nil
		}
		return m, cmd

	case screenReconstruction:
		next, cmd, submit, back := m.reconstruction.Update(msg)
		m.reconstruction = next
		if submit {
			r := m.reconstruction.Result()
			m.session.SubmitReconstruction(r)
			v := game.Score(m.session.Case, m.session.Log(), r)
			m.verdict = newVerdict(v, m.session.SavePath())
			m.screen = screenVerdict
			return m, nil
		}
		if back {
			m.screen = screenInterrogation
			return m, nil
		}
		return m, cmd

	case screenVerdict:
		next, cmd, done := m.verdict.Update(msg)
		m.verdict = next
		if done {
			_ = m.session.Close(context.Background())
			m.session = nil
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
	if m.helpOpen {
		return m.help.View()
	}
	switch m.screen {
	case screenSplash:
		return m.splash.View()
	case screenCasePicker:
		return m.casePicker.View()
	case screenBriefing:
		return m.briefing.View()
	case screenInterrogation:
		return m.interrogation.View()
	case screenReconstruction:
		return m.reconstruction.View()
	case screenVerdict:
		return m.verdict.View()
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

// openCase opens a save file, builds a driver via the factory,
// constructs a Session, and transitions to the interrogation screen.
// On any error, transitions to the error screen with errMsg set.
func (m model) openCase(c kase.Case) model {
	ctx := context.Background()
	savePath, err := newSavePath(c.ID)
	if err != nil {
		m.errMsg = err.Error()
		m.screen = screenError
		return m
	}
	driver, err := m.makeDriver(ctx, c, savePath)
	if err != nil {
		m.errMsg = err.Error()
		m.screen = screenError
		return m
	}
	session, err := game.NewSession(ctx, c, driver, game.DefaultBudget)
	if err != nil {
		_ = driver.Close()
		m.errMsg = err.Error()
		m.screen = screenError
		return m
	}
	m.session = session
	m.interrogation = newInterrogation(session)
	m.screen = screenInterrogation
	return m
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

// newBranchSavePath returns a save path for a branched session
// alongside the parent file. The timeline label is encoded into the
// filename so saves on disk are easy to relate.
func newBranchSavePath(caseID, parentTimeline string) (string, error) {
	saveDir, err := game.EnsureSaveDir()
	if err != nil {
		return "", err
	}
	sessionID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
	// e.g. streetlight-<ulid>-A.1.db
	return game.SavePath(saveDir, caseID, sessionID+"-"+parentTimeline+"-branch"), nil
}
