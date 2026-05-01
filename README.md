# hearsay

A memory-interrogation game in your terminal.

A witness saw something. She wants to help. She is sincerely trying to remember.
You ask questions. Some of what she says is real. Some of it is drift. Some of it
isn't hers at all. The truth was committed before you sat down. You have a finite
amount of her time. When she leaves, you reconstruct what happened, and the
hash chain proves what was true before any of your questions.

Built on [Starling](https://github.com/jerkeyray/starling).

## Install

```sh
go install github.com/jerkeyray/hearsay/cmd/hearsay@latest
```

Releases attach pre-built binaries for darwin/arm64, darwin/amd64, linux/amd64,
linux/arm64, and windows/amd64.

## Run

```sh
ANTHROPIC_API_KEY=sk-ant-... hearsay
```

Configuration is environment-only:

| Variable | Default | Notes |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | — | Default provider. Tunes the dryness register best. |
| `OPENAI_API_KEY` | — | Falls back automatically; or set `PROVIDER=openai`. |
| `PROVIDER` | autodetect | `anthropic` or `openai` to pin. |
| `MODEL` | `claude-sonnet-4-6` (anthropic) / `gpt-4o-mini` (openai) | Override per provider. |
| `OPENAI_BASE_URL` | — | OpenAI-compatible gateway (Groq, etc.). |
| `HEARSAY_HOME` | `~/.hearsay` | Where save files live. |
| `HEARSAY_DEBUG` | unset | `1` enables structured slog → `$HEARSAY_HOME/debug.log` (info). `2` is debug-level. |

If no key is set, the binary runs with a canned-line stub witness so you can
explore the loop. The dryness brief and recall semantics need a real LLM.

## How to play

1. Splash → **new case** → briefing.
2. Pick **what to ask** and **how**:
   - **directly** — the witness's most-readily-believed version.
   - **the moment before** — shifts the anchor; suppressed memories sometimes surface.
   - **how do you know** — forces a source; implants thin, confabulations turn circular.
   - **push back** — challenge; real holds, drift drifts further, implants double down.
   - **circle back later** — mark the topic to ask again.
3. Press `r` to **rewind** to a prior turn in the same timeline; `b` to **branch**
   into a new one. The session clock counts down as the witness gets tired.
4. Press `i` to open the **inspector** — a read-only view of every event in
   the SQLite save file.
5. Press `d` when you're done. The reconstruction form asks what actually happened.
6. The verdict screen reveals the locked truth and classifies your errors. Press
   `v` to verify the BLAKE3 hash chain — the truth was committed before you
   sat down, and the chain proves it.

`?` opens help anywhere.

## Architecture

```
cmd/hearsay/         entrypoint, env-driven provider selection
cases/<name>/        per-case content (one file = one case, no engine changes)
internal/
  ui/                bubble tea models for each screen
  game/              session state, scoring, rewind/branch, verify
  witness/           prompt, recall + note_demeanor tools, live + stub drivers
  kase/              Case / Topic / Belief / Form / Rubric data types
```

The architecture goal is that adding a new case is writing one Go file in
`cases/<name>/case.go`. The engine never imports a specific case.

## Watching the API calls

Three angles:

1. **In-game inspector.** Press `i` during interrogation. Every Starling
   event is a row: `RunStarted`, `TurnStarted`, `ToolCallScheduled`,
   `ToolCallCompleted`, `AssistantMessageCompleted`, `RunCompleted`. `enter`
   expands a row to show seq, runID, timestamp, prev-hash, payload size.
2. **Streaming text log.** Run with `HEARSAY_DEBUG=1` (info) or `2` (debug)
   and tail the log in another terminal:
   ```sh
   HEARSAY_DEBUG=2 ANTHROPIC_API_KEY=... hearsay
   # in another shell:
   tail -f ~/.hearsay/debug.log
   ```
   You'll see Starling's per-step output: turn boundaries, tool calls,
   token usage, budget tracking, terminal events.
3. **Direct SQL.** The save file is plain SQLite:
   ```sh
   sqlite3 ~/.hearsay/saves/streetlight-*.db \
     "SELECT seq, kind, length(payload) FROM events ORDER BY seq;"
   ```
   Payloads are canonical CBOR; decode with the `event` package
   from Starling if you want the actual provider request/response.

## Saves

One SQLite file per session at `~/.hearsay/saves/<case>-<sessionID>.db`. Branches
are sibling files. Rewinds keep all events on disk so the verify chain stays
honest; the in-memory log is the "current timeline" the verdict scores against.

## Cost

Default budget per case: 3,000 output tokens, $0.05. The session clock maps
1000 tokens to 1 minute, so the witness arrives offering you 3:00 — about
10–20 asks in live mode, ~15 in stub. Anthropic prompt caching keeps the
static system prompt cheap across turns; expect well under $0.05 per
playthrough at default settings. Bump `DefaultBudget` in `internal/game/session.go`
if you want a longer interrogation.

## Status

This is v0.x. The dryness brief is the highest-leverage file in the repo;
expect it to evolve from playtests. The five providers Starling supports
work in principle; the dryness register has been tuned against Anthropic
Sonnet 4.6 and may need per-provider adjustments elsewhere.

## Layout for development

`go.mod` has `replace github.com/jerkeyray/starling => ../starling` for
co-development. Clone the repos as siblings:

```
code/
├── hearsay/
└── starling/
```

CI (`.github/workflows/ci.yml`) and release (`.github/workflows/release.yml`)
mirror this — both check out `jerkeyray/hearsay` and `jerkeyray/starling`
into sibling directories on the runner so the `replace` resolves.

## Releasing

1. Tag this repo: `git tag v0.1.0 && git push --tags`.
2. The release workflow builds and uploads binaries for darwin/arm64,
   darwin/amd64, linux/amd64, linux/arm64, windows/amd64.

For `go install github.com/jerkeyray/hearsay/cmd/hearsay@latest` to work
for end users (not just sibling-checkout developers), Starling must also
be tagged and the `replace` line must be removed from `go.mod` in favor
of a normal `require` against the tagged Starling version.

## License

See [LICENSE](LICENSE).
