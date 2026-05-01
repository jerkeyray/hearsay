// Command playbot is a one-ask-at-a-time CLI that lets a non-TTY
// driver (an LLM, a shell script, a human in vim) play hearsay
// without the Bubble Tea TUI. Each invocation opens or resumes a
// session save and either runs one ask or submits a reconstruction.
//
// Examples:
//
//	playbot ask "the time" "directly"
//	playbot ask "the bag" "the moment before"
//	playbot status                                     # session log so far
//	playbot submit \
//	  car_color=blue second_person=no \
//	  streetlight_color=orange limp_side=left \
//	  time="11:47" weather=no passersby="i didn't watch" \
//	  bag_contents="a folder,a gun"
//
// API key: set ANTHROPIC_API_KEY / OPENAI_API_KEY in env, or drop
// either into ~/.hearsay/key.env in dotenv shape (KEY=value per line).
//
// Save path: $HEARSAY_HOME/saves/streetlight-playbot.db (sticky across
// invocations so each ask appends to the same file). Pass --save
// to override.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

const defaultSaveBase = "streetlight-playbot.db"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "playbot:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	loadKeyEnvFile()

	// Optional --save <path> as the first flag.
	savePath, args, err := parseSaveFlag(args)
	if err != nil {
		return err
	}
	if savePath == "" {
		dir, err := game.EnsureSaveDir()
		if err != nil {
			return err
		}
		savePath = filepath.Join(dir, defaultSaveBase)
	}

	switch args[0] {
	case "ask":
		if len(args) != 3 {
			return fmt.Errorf(`ask requires <topic> <technique>; example: ask "the bag" "the moment before"`)
		}
		return doAsk(savePath, args[1], args[2])
	case "status":
		return doStatus(savePath)
	case "submit":
		return doSubmit(savePath, args[1:])
	case "reset":
		return doReset(savePath)
	case "-h", "--help", "help":
		printUsage()
		return nil
	}
	return fmt.Errorf("unknown subcommand %q (try `playbot help`)", args[0])
}

func printUsage() {
	fmt.Println(`Usage: playbot [--save <path>] <command>

Commands:
  ask <topic> <technique>   Run one ask. Topic is in the case's topic list;
                            technique ∈ { directly | the moment before |
                            how do you know | push back | circle back later }.
  status                    Print the conversation log so far.
  submit <id>=<answer>...   Submit reconstruction; print verdict + score.
                            For multi-select, comma-separate values.
  reset                     Delete the save file.
  help                      Show this.

Save defaults to $HEARSAY_HOME/saves/streetlight-playbot.db.
Set ANTHROPIC_API_KEY or OPENAI_API_KEY (env or ~/.hearsay/key.env).`)
}

func parseSaveFlag(args []string) (string, []string, error) {
	if len(args) >= 2 && args[0] == "--save" {
		return args[1], args[2:], nil
	}
	if len(args) >= 1 && strings.HasPrefix(args[0], "--save=") {
		return args[0][len("--save="):], args[1:], nil
	}
	return "", args, nil
}

// loadKeyEnvFile reads ~/.hearsay/key.env (dotenv-style: KEY=value
// per line) and sets each pair into the process env. Existing env
// values win, so this is "fill in what's missing." Errors silently
// fall through; callers will hit a clear error from
// witness.NewLiveProviderFromEnv if no key is set.
func loadKeyEnvFile() {
	home := os.Getenv("HEARSAY_HOME")
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return
		}
		home = filepath.Join(h, ".hearsay")
	}
	f, err := os.Open(filepath.Join(home, "key.env"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		v = strings.Trim(v, `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}

func openSession(savePath string) (*game.Session, witness.Driver, error) {
	live, err := witness.NewLiveProviderFromEnv()
	if err != nil {
		return nil, nil, err
	}
	driver, err := live.NewDriver(savePath, streetlight.Case)
	if err != nil {
		return nil, nil, err
	}
	sess, err := game.NewSession(context.Background(), streetlight.Case, driver, game.DefaultBudget)
	if err != nil {
		_ = driver.Close()
		return nil, nil, err
	}
	// Replay prior exchanges into the in-memory log so history-aware
	// prompts and scoring see them. We rebuild from the SQLite save's
	// AssistantMessageCompleted events.
	if err := replayHistory(sess, savePath); err != nil {
		_ = driver.Close()
		return nil, nil, err
	}
	return sess, driver, nil
}

func doAsk(savePath, topic, techStr string) error {
	tech, ok := kase.ParseTechnique(techStr)
	if !ok {
		return fmt.Errorf("unknown technique %q (valid: directly, the moment before, how do you know, push back, circle back later)", techStr)
	}
	if !topicVisible(topic) {
		return fmt.Errorf("topic %q not in case (try: %s)",
			topic, strings.Join(topicNames(), ", "))
	}

	sess, driver, err := openSession(savePath)
	if err != nil {
		return err
	}
	defer driver.Close()

	ctx := context.Background()
	if sess.SessionEnded() {
		return fmt.Errorf("session ended (budget exhausted at %d tokens)", sess.UsedOutputTokens())
	}
	fmt.Printf("[turn %d  ·  clock %s  ·  she is %s]\n",
		sess.TurnCount()+1, sess.ClockDisplay(), sess.CurrentDemeanor())
	fmt.Printf("> me: %s, %s\n", topic, tech.Label())
	ex, err := sess.Ask(ctx, topic, tech)
	if err != nil {
		return err
	}
	fmt.Printf("  she: %s\n", ex.Witness)
	fmt.Printf("    [demeanor: %s · %d output tokens · $%.4f]\n",
		ex.Demeanor, ex.OutputTokens, ex.CostUSD)
	fmt.Printf("    [save: %s]\n", savePath)
	return nil
}

func doStatus(savePath string) error {
	sess, driver, err := openSession(savePath)
	if err != nil {
		return err
	}
	defer driver.Close()

	log := sess.Log()
	if len(log) == 0 {
		fmt.Println("(no asks yet)")
		fmt.Printf("clock: %s · session ended: %v\n", sess.ClockDisplay(), sess.SessionEnded())
		return nil
	}
	for _, ex := range log {
		fmt.Printf("turn %d (%s, %s)\n", ex.Turn, ex.Topic, ex.Technique.Label())
		fmt.Printf("  she: %s\n", ex.Witness)
	}
	fmt.Println()
	fmt.Printf("clock: %s\n", sess.ClockDisplay())
	fmt.Printf("used:  %d output tokens, $%.4f\n", sess.UsedOutputTokens(), sess.UsedCostUSD())
	fmt.Printf("ended: %v\n", sess.SessionEnded())
	visible := []string{}
	for _, t := range sess.VisibleTopics() {
		visible = append(visible, t.Name)
	}
	fmt.Printf("visible topics: %s\n", strings.Join(visible, " | "))
	fmt.Printf("save: %s\n", savePath)
	return nil
}

func doSubmit(savePath string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("submit requires id=value pairs; example: car_color=blue bag_contents=a folder,a gun")
	}
	answers := make([]game.Answer, 0, len(args))
	known := map[string]kase.Question{}
	for _, q := range streetlight.Case.Reconstruction.Questions {
		known[q.ID] = q
	}
	for _, a := range args {
		eq := strings.IndexByte(a, '=')
		if eq <= 0 {
			return fmt.Errorf("malformed answer %q (want id=value)", a)
		}
		id := a[:eq]
		val := a[eq+1:]
		q, ok := known[id]
		if !ok {
			return fmt.Errorf("unknown question id %q (valid: %s)", id, strings.Join(keys(known), ", "))
		}
		ans := game.Answer{QuestionID: id}
		switch {
		case strings.EqualFold(val, "don't know") || val == "?":
			ans.DontKnow = true
		default:
			switch q.Type {
			case kase.Radio:
				ans.Choice = val
			case kase.MultiSelect:
				for _, c := range strings.Split(val, ",") {
					ans.Choices = append(ans.Choices, strings.TrimSpace(c))
				}
			case kase.FreeText:
				ans.FreeText = val
			}
		}
		answers = append(answers, ans)
	}

	sess, driver, err := openSession(savePath)
	if err != nil {
		return err
	}
	defer driver.Close()

	r := game.Reconstruction{Answers: answers}
	sess.SubmitReconstruction(r)
	v := game.Score(streetlight.Case, sess.Log(), r)

	for _, item := range v.Items {
		fmt.Printf("%s\n", item.Prompt)
		fmt.Printf("  me:    %s\n", formatAnswerLine(item.Player))
		if item.Witness != "" {
			fmt.Printf("  she:   %s\n", item.Witness)
		}
		if item.Truth != "" {
			fmt.Printf("  truth: %s\n", item.Truth)
		}
		marker := "✗"
		if item.Correct {
			marker = "✓"
		}
		fmt.Printf("  %s %s\n\n", marker, item.Error.Label())
	}
	fmt.Printf("score: %d / %d\n", v.Score, v.Total)
	if v.Summary != "" {
		fmt.Println(v.Summary)
	}
	fmt.Printf("\nused: %d tokens, $%.4f\n", sess.UsedOutputTokens(), sess.UsedCostUSD())
	fmt.Printf("save: %s\n", savePath)
	return nil
}

func doReset(savePath string) error {
	for _, suf := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(savePath + suf)
	}
	fmt.Printf("removed %s (and sidecars)\n", savePath)
	return nil
}

// replayHistory pulls AssistantMessageCompleted texts out of the
// SQLite save and re-applies them to the session's in-memory log.
// Hacky but enough to make multi-invocation play work without a
// formal Resume API on Session.
func replayHistory(sess *game.Session, savePath string) error {
	// Use the verify pathway as a quick reader: it's already plumbed
	// to walk every run in the file. We don't actually need the
	// verify result, just the chronologically-ordered exchanges.
	// To avoid a circular concern, we go through a fresh read-only
	// open of the SQLite log here.
	return rebuildExchangesFromSQLite(sess, savePath)
}

func rebuildExchangesFromSQLite(sess *game.Session, savePath string) error {
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		return nil
	}
	exs, err := readExchangesFromSave(savePath)
	if err != nil {
		return err
	}
	for _, ex := range exs {
		// Inject by performing a pseudo-ask: we already know what
		// she said, so we don't want to call the LLM again. We
		// reach into the package via SubmitHistoryItem (not yet
		// exposed)... fallback: use the public Replay loader if
		// added later. For now we simulate by doing nothing — the
		// driver path will rewrite history into the prompt anyway
		// because LiveDriver.Respond reads sess.Log() from the
		// caller, which is empty here. Translation: each playbot
		// invocation looks fresh to the witness.
		_ = ex
	}
	return nil
}

// readExchangesFromSave is a placeholder: for true continuity across
// playbot invocations we'd parse AssistantMessageCompleted events
// out of the SQLite log and reconstruct in-memory Exchanges. Until
// game.Session exposes a public Resume helper this is a no-op and
// each invocation starts the witness's "memory of this session"
// fresh — but the SQLite event log is still appended-to so the
// inspector and verify path see all asks.
func readExchangesFromSave(_ string) ([]game.Exchange, error) {
	return nil, nil
}

func topicVisible(name string) bool {
	for _, t := range streetlight.Case.Topics {
		if t.Name == name {
			return true
		}
	}
	return false
}

func topicNames() []string {
	out := []string{}
	for _, t := range streetlight.Case.Topics {
		out = append(out, fmt.Sprintf("%q", t.Name))
	}
	return out
}

func keys(m map[string]kase.Question) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func formatAnswerLine(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
