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

## Saves

One SQLite file per session at `~/.hearsay/saves/<case>-<sessionID>.db`. Branches
are sibling files. Rewinds keep all events on disk so the verify chain stays
honest; the in-memory log is the "current timeline" the verdict scores against.

## Cost

Default budget per case: 50,000 output tokens, $0.40. Anthropic prompt caching
keeps the static system prompt cheap across turns. Expect $0.10–$0.40 per
playthrough at default settings.

## Status

This is v0.x. The dryness brief is the highest-leverage file in the repo;
expect it to evolve from playtests. The five providers Starling supports
work in principle; the dryness register has been tuned against Anthropic
Sonnet 4.6 and may need per-provider adjustments elsewhere.

## Releasing

CI (`.github/workflows/ci.yml`) and release (`.github/workflows/release.yml`)
assume Starling is publicly resolvable as a Go module. The current `go.mod`
uses a local `replace` directive pointing at `/Users/jerkeyray/code/starling`
during co-development. Before the first tagged release:

1. Tag a Starling version and push it.
2. Drop the `replace` line from `go.mod` and pin a normal `require` against
   the tagged version.
3. Tag this repo (`git tag v0.1.0 && git push --tags`); release workflow
   builds and uploads binaries for the five target platforms.

## License

See [LICENSE](LICENSE).
