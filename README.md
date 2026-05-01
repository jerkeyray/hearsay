```
 _                                 
| |__   ___  __ _ _ __ ___  __ _  _   _ 
| '_ \ / _ \/ _` | '__/ __|/ _` || | | |
| | | |  __/ (_| | |  \__ \ (_| || |_| |
|_| |_|\___|\__,_|_|  |___/\__,_| \__, |
                                  |___/ 
```

> a memory-interrogation game in your terminal

A witness saw something. She wants to help. She is sincerely trying to remember.
You ask questions. Some of what she says is real. Some of it is drift. Some of it
isn't hers at all. The truth was committed before you sat down. You have a finite
amount of her time. When she leaves, you reconstruct what happened, and the
hash chain proves what was true before any of your questions.

Built on [Starling](https://github.com/jerkeyray/starling).

---

## install

```sh
go install github.com/jerkeyray/hearsay/cmd/hearsay@latest
```

## run

```sh
ANTHROPIC_API_KEY=sk-ant-... hearsay
```

If no key is set, the binary runs with a canned-line stub witness so you can
explore the loop. The dryness brief and recall semantics need a real LLM.

Configuration is environment-only:

| variable | default | notes |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | — | default provider; tunes the dryness register best |
| `OPENAI_API_KEY` | — | falls back automatically; or `PROVIDER=openai` |
| `PROVIDER` | autodetect | `anthropic` or `openai` to pin |
| `MODEL` | `claude-sonnet-4-6` / `gpt-4o-mini` | override per provider |
| `OPENAI_BASE_URL` | — | OpenAI-compatible gateway (Groq, etc.) |
| `HEARSAY_HOME` | `~/.hearsay` | where save files live |
| `HEARSAY_DEBUG` | unset | `1` writes slog to `$HEARSAY_HOME/debug.log`; `2` is debug-level |

---

## how to play

1. splash → **new case** → briefing.
2. pick **what to ask** and **how**:
   - **directly** — her most-readily-believed version.
   - **the moment before** — shifts the anchor; suppressed memories sometimes surface.
   - **how do you know** — forces a source; implants thin, confabulations turn circular.
   - **push back** — challenge; real holds, drift drifts further, implants double down.
   - **circle back later** — mark the topic to ask again.
3. `r` rewinds to a prior turn in the same timeline; `b` branches into a new one.
   the session clock counts down as the witness gets tired.
4. `i` opens the inspector — every event in the SQLite save, read-only.
5. `d` when you're done. the reconstruction form asks what actually happened.
6. the verdict reveals the locked truth and classifies your errors.
   `v` verifies the BLAKE3 hash chain — the truth was committed before you sat
   down, and the chain proves it.

`?` opens help anywhere.

---

## watching the api calls

four angles, in order of ergonomic depth:

1. **in-game inspector** — `i` during interrogation. every Starling event as a row.
2. **Starling's web inspector** — same SQLite log, fuller view, payloads decoded,
   Replay button that re-runs any recorded run against a fresh agent.
   ```sh
   hearsay inspect ~/.hearsay/saves/streetlight-<sessionID>.db
   ```
   Opens HTTP on `127.0.0.1:<random>`. With an API key set, Replay works; without
   one it's read-only.
3. **streaming text log**:
   ```sh
   HEARSAY_DEBUG=2 ANTHROPIC_API_KEY=... hearsay
   tail -f ~/.hearsay/debug.log     # in another terminal
   ```
4. **direct SQL** — saves are plain SQLite, payloads are canonical CBOR.

---

## architecture

```
cmd/hearsay/      entrypoint, env-driven provider selection
cmd/playbot/      non-TTY CLI for scripted / agent-driven play
cases/<name>/     per-case content; one file = one case, no engine changes
internal/
  ui/             bubble tea models for each screen
  game/           session state, scoring, rewind/branch, verify
  witness/        prompt, recall + note_demeanor tools, live + stub drivers
  kase/           Case / Topic / Belief / Form / Rubric data types
```

Adding a case is writing one Go file in `cases/<name>/case.go`. The engine never
imports a specific case.

## saves

One SQLite file per session at `~/.hearsay/saves/<case>-<sessionID>.db`. Branches
are sibling files. Rewinds keep all events on disk so the verify chain stays
honest; the in-memory log is the "current timeline" the verdict scores against.

## cost

Default budget: **3,000 output tokens, $0.05** per case. The session clock maps
1000 tokens to 1 minute — the witness arrives offering you 3:00, about 10–20
asks in live mode. Bump `DefaultBudget` in `internal/game/session.go` if you
want a longer interrogation.

---

## development

`go.mod` uses `replace github.com/jerkeyray/starling => ../starling` for
co-development. Clone the repos as siblings:

```
code/
├── hearsay/
└── starling/
```

For `go install ...@latest` to work for end users, Starling has to be tagged and
the `replace` line dropped in favor of a normal `require` against the tagged
version. Then `git tag v0.1.0 && git push --tags`, and build release binaries
locally with `go build -trimpath -ldflags '-s -w' ./cmd/hearsay` per target.

## status

v0.x. The dryness brief is the highest-leverage file in the repo; expect it to
evolve from playtests. Anthropic Sonnet 4.6 is the tuned default; other
providers may need per-provider register tweaks.

## license

[LICENSE](LICENSE).
