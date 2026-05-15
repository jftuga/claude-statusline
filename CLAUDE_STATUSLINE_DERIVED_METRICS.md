# Derived Metrics Ideas

Metrics that could be calculated from the existing JSON schema fields.

## $/min
`total_cost_usd / (total_duration_ms / 60000)`

Cost burn rate. Compare across models/effort levels to decide if high-effort Opus is worth it vs Sonnet.

## $/hr
`total_cost_usd / (total_duration_ms / 3600000)`

Same as $/min but easier to reason about for longer sessions.

## Cache hit %
`cache_read / (cache_read + cache_creation + input_tokens)`

How well prompt caching is working. Low number on a long session may indicate a context structure problem.

## API wait %
`total_api_duration_ms / total_duration_ms`

Fraction of wall clock spent waiting on API calls vs tool execution and file reads. More curiosity than actionable.

## Lines/dollar
`(total_lines_added + total_lines_removed) / total_cost_usd`

Bang for your buck. Noisy -- a large refactor isn't necessarily more valuable than a 3-line bug fix.

## Session token total
`total_input_tokens + total_output_tokens`

Cumulative tokens consumed in the session. Unlike the existing token counter (`⟐ 16k/200k`) which shows current context window usage and drops after compaction, this never goes down. Partially redundant with t/m (same numerator) and cost (roughly proportional). Main value is raw magnitude: "this session has chewed through 2.4M tokens total."

---

# Persistent Stats (Cross-Session)

Persisting stats to a local file and reading on each invocation would enable cumulative metrics.

## What persistence unlocks

- **Daily/weekly spend** -- aggregate `total_cost_usd` across sessions. Most actionable: "I've spent $14 today across 6 sessions."
- **Per-model cost breakdown** -- "Opus: $42/week, Sonnet: $8/week." Informs model selection habits.
- **Session count/average duration** -- are you running 3 long sessions or 20 short ones?
- **Rate limit history** -- how often you're crossing 80% on the 5h window. Pattern detection: "I always hit limits around 2pm."
- **Trend lines** -- is your daily spend increasing week over week?

## Concerns

- **Performance** -- the whole point of this binary is 1.8ms. File I/O (read + parse + write) on every invocation adds measurable latency, and the statusline runs frequently.
- **Concurrency** -- multiple Claude Code sessions running simultaneously means file locking or corruption risk.
- **Scope creep** -- this tool is a renderer (stdin in, ANSI out). Persistence turns it into a metrics collector. Those are different tools.
- **Deduplication** -- distinguishing "same session, updated stats" from "new session" requires keying on `session_id`.

Daily/weekly spend is the most compelling candidate. A separate tool that reads transcript files or a post-session hook might be a cleaner fit than bolting state onto a status line renderer.
