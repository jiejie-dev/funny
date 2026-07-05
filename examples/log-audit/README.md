# funny log-audit

A small but real access-log auditing tool, written entirely in
[funny](../../README.md) (the "v2" scripting language implemented in this
repository). It reads a pipe-delimited HTTP access log, parses and
validates every line, computes error-rate/latency statistics, renders a
human-readable report, cryptographically signs that report so it can be
verified later, and optionally pings a live upstream endpoint as a health
check.

It exists for two reasons at once:

1. **It is a genuinely useful little tool.** Point it at a real access log
   (`LOG_AUDIT_FILE=/var/log/nginx/access.log`, reformatted to the pipe
   layout below) and it tells you your error rate, your slowest
   endpoints, and whether you've breached a simple SLO — the kind of
   report you'd actually want after a deploy or an incident.
2. **It is a tour of the language.** Between the six `.fn` files here,
   essentially every implemented feature of funny gets exercised in
   context, not in isolation — see the [feature map](#feature-map) below.

## Quick start

```bash
# from the repository root, build the CLI once:
go build -o funny ./cmd/funny

# then, from this directory:
cd examples/log-audit
../../funny run main.fn
```

That's it — `main.fn` defaults to reading the bundled `sample.log`, so it
runs with no configuration. Expected output looks like:

```
funny log-audit
input file: sample.log

== Status code breakdown ==
  2xx success          37
  3xx redirect          1
  4xx client error      7
  5xx server error      3

== Summary ==
  total requests          48
  errors (4xx/5xx)        10
  error rate              20.8%
  avg response time       284.6 ms
  stddev response         414.8 ms
  slowest request     /api/products/search (2360 ms)

== Top 3 slowest requests ==
  1. GET   /api/products/search           2360 ms
  2. GET   /api/reports/daily             1883 ms
  3. GET   /api/reports/daily              975 ms

== Health verdict ==
  FAIL  error rate 20.8% exceeds 15% threshold

== Data quality ==
  2 malformed line(s) skipped:
    - malformed log line: malformed-entry-missing-fields|GET
    - malformed log line: 1735809999|not-an-ip-and-bad-status|GET|/api/users|abc|xyz

== Signed report ==
  body:     total=48;errors=10;error_rate=0.2083;avg_ms=284.65;stddev_ms=414.80
  sha256:   e831e91e27e48b233fe76c9c15d4ed47b70d8783dc0da0e717ad0a4e53cba9c5
  token:    eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyYXdfY2xhaW1zIjoi...
  verify with correct secret: true
  verify with wrong secret:   false

== Upstream health check (optional, network required) ==
  target: https://httpbin.org/status/200
  result: reachable (0 bytes)
```

### Configuration (environment variables)

All optional — every one has a working default so the tool runs out of
the box.

| Variable            | Default                          | Meaning                                                      |
|----------------------|-----------------------------------|---------------------------------------------------------------|
| `LOG_AUDIT_FILE`     | `sample.log`                      | Path to the access log to audit.                              |
| `AUDIT_SECRET`       | `dev-secret-change-me`            | HMAC secret used to sign/verify the report. **Set this for real use.** |
| `HEALTH_CHECK_URL`   | `https://httpbin.org/status/200`  | URL pinged by the final health-check section.                 |

Examples:

```bash
# audit a real log file, with your own signing secret
LOG_AUDIT_FILE=/var/log/nginx/access.log.funny AUDIT_SECRET="$(openssl rand -hex 32)" \
  ../../funny run main.fn

# point the health check at your own service, or skip the network entirely
HEALTH_CHECK_URL="https://your-service.internal/healthz" ../../funny run main.fn
```

### The log format

`sample.log` uses a deliberately simple pipe-delimited format, one
request per line:

```
<unix_timestamp>|<ip>|<METHOD>|/<path>|<status>|<response_ms>
```

e.g. `1735808441|203.0.113.5|GET|/api/products/search|200|79`. This isn't
a real access-log format (nginx/Apache combined log is unquoted-hostile
and needs a real parser) — it exists to put `regex_match`/`str_split`
front and center for the walkthrough below without fighting quoting
rules. `sample.log` includes two intentionally malformed lines, so the
"Data quality" section of the report always has something to show.

## Project layout

```
log-audit/
├── models.fn      shared structs: LogEntry, StatusBucket, Stats
├── parser.fn       line parsing (regex_match, str_split) -> Result[LogEntry, str]
├── stats.fn        aggregation: buckets, error rate, avg/stddev, top-N slowest
├── security.fn     sha256 checksum + jwt_encode/jwt_decode signing & verification
├── net_check.fn     optional http_get health check (imported with `as net`)
├── workflow.fn      the same audit re-expressed as a `plan` (agent-protocol) block
├── sample.log       51 lines of synthetic access-log data (incl. 2 malformed lines)
└── main.fn          entry point: wires every module together, prints the report
```

`main.fn` is the thing you actually run. `workflow.fn` is a separate,
parallel demonstration of funny's *other* execution model (see
["Plans" below](#plans-agent-protocol-workflowfn)) and isn't wired into
`main.fn`'s pipeline.

## Feature map

Every row below is exercised by real code in this directory, not a
toy snippet — click through to see it in context.

| Feature | Where |
|---|---|
| `struct` declarations | `models.fn` — `LogEntry`, `StatusBucket`, `Stats` |
| `fn` / `pub fn` | every module — private helpers vs. cross-module API |
| Typed parameters & return types (`int`, `float`, `str`, `bool`, `nil`, `list[T]`, struct types) | throughout, e.g. `stats.fn`'s `summarize(entries: list[LogEntry]) -> Stats` |
| Plain `import "./x.fn"` | `main.fn` importing `models.fn`/`parser.fn`/`stats.fn`/`security.fn` |
| Aliased `import "./x.fn" as name` | `main.fn` importing `net_check.fn` as `net`, called as `net.check_endpoint(...)` |
| `let` with and without type annotations, incl. empty-literal annotations (`let xs: list[T] = []`) | `stats.fn`'s `top_n_slowest` |
| Assignment / reassignment | loop accumulators throughout `stats.fn`, `parser.fn` |
| `if` / `elif` / `else` | `stats.fn`'s `count_by_status_class`, `main.fn`'s `print_health_verdict` |
| `for x in list` | every aggregation function in `stats.fn` |
| `while` | `stats.fn`'s `top_n_slowest` (bounded selection loop) |
| `and` / `or` / `not` | `parser.fn`'s validation, `stats.fn`'s `while taken < n and len(pool) > 0` |
| Lists: literals, indexing, `len()`, `append()` | `parser.fn`, `stats.fn` throughout |
| Struct literals & field access | `LogEntry(...)`, `StatusBucket(...)`, `Stats(...)` construction and `.field` reads everywhere |
| `Result[T, E]` + `?` operator | `parser.fn`'s `parse_log_line` (`return err(...)?` / `return ok(entry)?`), `.tag`/`.val` reads in `main.fn` |
| f-strings, incl. format specs (`:<N`, `:>N`, `:.Nf`, `:.N%`) | `main.fn`'s report-printing functions |
| Builtins: `len`, `to_str`, `to_int`, `to_float`, `sqrt`, `regex_match`, `str_split`, `append`, `env_get`, `file_exists`, `file_read`, `sha256`, `jwt_encode`, `jwt_decode`, `http_get` | see the table in [Builtins used](#builtins-used) |
| `plan` / `meta` (agent protocol): `tool`/`transform`/`branch`/`guard`/`parallel`/`delay` steps, `retry`+`backoff`, `timeout` | `workflow.fn` (see below) |
| CLI tooling: `run`, `fmt`, `ast`, `describe`, `disasm` | see [Exploring with the CLI](#exploring-with-the-cli) |

### Builtins used

| Builtin | Used in | Purpose |
|---|---|---|
| `len` | `parser.fn`, `stats.fn`, `main.fn` | list/string length |
| `to_str` / `to_int` / `to_float` | `parser.fn`, `stats.fn` | field conversion; int→float division for rates/averages |
| `sqrt` | `stats.fn` | population standard deviation of response times |
| `regex_match` | `parser.fn` | validating a line's shape before splitting it |
| `str_split` | `parser.fn` | splitting a line/file into fields/lines |
| `append` | `parser.fn`, `stats.fn` | building up result lists (non-mutating: returns a new list) |
| `env_get` | `main.fn`, `net_check.fn` | reading `LOG_AUDIT_FILE`/`AUDIT_SECRET`/`HEALTH_CHECK_URL` |
| `file_exists` / `file_read` | `main.fn` | loading the target log, with a friendly error if it's missing |
| `sha256` | `security.fn` | fingerprinting the report body |
| `jwt_encode` / `jwt_decode` | `security.fn` | HMAC-signing and verifying the report |
| `http_get` | `net_check.fn` | the optional upstream health check |

## Plans (agent protocol): `workflow.fn`

funny has a second, separate execution model alongside the ordinary
imperative one: a `plan` block, made of named `step`s with kinds
(`tool`, `transform`, `branch`, `guard`, `parallel`, `delay`) and
per-step `retry`/`backoff`/`timeout` options. `workflow.fn` re-expresses
a miniature version of this same audit as a plan, to show every step
kind and option funny's plan DSL currently supports in one place.

**This file is not run by `funny run`.** Plan/step bodies aren't
compiled or type-checked by the bytecode VM at all (`funny run`
silently no-ops a top-level `plan` block); they execute on a *separate*
tree-walking engine (`internal/agent.Engine`) that is currently only
exercised from Go code (see `internal/agent/engine_test.go`) and isn't
yet wired up to any CLI subcommand or to the `funny-mcp` server's
`run_skill` tool. That engine also only implements funny's small core
builtin set (`print`/`println`/`len`/`to_str`/`to_int`/`type_of`/`ok`/`err`),
not the extended stdlib (`file_*`/`http_get`/`regex_match`/`sha256`/
`jwt_*`) `main.fn` relies on — which is why `workflow.fn` operates on a
small embedded batch of status codes instead of reading `sample.log`
directly.

What you *can* do with it today:

```bash
# print the plan's static structure (step names) as JSON
../../funny describe workflow.fn

# print the full step tree - kinds, retry/backoff/timeout, guard/parallel bodies
../../funny ast workflow.fn

# format it
../../funny fmt workflow.fn
```

...and, over LSP, the custom `funny/planGraph` request turns the same
step tree into a renderable node/edge graph for editor tooling.

## Exploring with the CLI

All commands below assume you're in this directory and have built
`funny` at the repository root (`go build -o funny ./cmd/funny`).

```bash
# run the audit
../../funny run main.fn

# format every file in place (round-trips cleanly - try it, then `git diff`)
for f in *.fn; do ../../funny fmt -w "$f"; done

# inspect the parsed AST as JSON, e.g. to see how `parser.fn`'s `?`
# operator or `main.fn`'s f-string format specs actually parse
../../funny ast parser.fn

# disassemble compiled bytecode, e.g. to see how a `for` loop or a
# struct literal actually lowers to VM instructions
../../funny disasm stats.fn

# print plan/metadata (empty for main.fn/parser.fn/etc., since they have
# no plan block; see workflow.fn above for a non-empty example)
../../funny describe workflow.fn
```

## Testing malformed input

`sample.log` intentionally includes two unparseable lines so the "Data
quality" report section always has content:

```
malformed-entry-missing-fields|GET
1735809999|not-an-ip-and-bad-status|GET|/api/users|abc|xyz
```

`parse_log`/`parse_log_errors` (in `parser.fn`) split log parsing into
"give me the good entries" and "give me the reasons for the bad ones" as
two separate `pub fn`s over the same `Result`-returning `parse_log_line`,
so a single malformed line degrades the report instead of crashing the
whole audit — try appending your own malformed line to `sample.log` (or
pointing `LOG_AUDIT_FILE` at a file full of them) and re-running to see
it show up in the "Data quality" section.

## Known limitations (of the language, not this example)

These are real, current gaps in funny itself, documented here rather
than worked around, so this example doesn't overstate what's
implemented:

- No `match` statement yet (parses to `E1003`) — `stats.fn`'s status-code
  bucketing uses an `if`/`elif`/`else` chain instead.
- No `sort()` builtin — `top_n_slowest` (`stats.fn`) implements "top N"
  via repeated selection instead of sort-then-slice.
- No date/time parsing builtin — timestamps are carried as raw
  `int` (Unix seconds) rather than a real datetime type.
- `plan` blocks don't run under `funny run`/`funny-mcp`'s `run_skill`
  yet (see [Plans](#plans-agent-protocol-workflowfn) above) — hence
  `workflow.fn` being a parallel, non-executed demonstration.
