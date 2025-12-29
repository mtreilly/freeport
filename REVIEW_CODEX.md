# Freeport Review (Codex)

## Idea in one sentence
Make local dev port usage obvious and make the common fix (“what’s on 3000?” / “free 3000”) one command, with optional automation for choosing a different port.

## Goals that look good (keep)
- Zero/low adoption friction: works without app integration.
- Fast visibility: `list`/`who` is a better `lsof` for humans and tools.
- Safe, scriptable actions: no interactive prompts; deterministic exit codes.
- Optional “smart wrapper” later, not required for baseline value.

## What still needs tightening (agent-friendly focus issues)
- “Passive daemon + SQLite registry” as the starting point adds operational surface area (daemon lifecycle, IPC, upgrades, perms) without improving correctness over querying the OS on-demand.
- “Zombie ports / TIME_WAIT cleanup” is easy to misstate: TIME_WAIT is not a listening process; user pain is usually “a process is still LISTENing” (or IPv6/IPv4 binding mismatch), which a tool can identify and optionally terminate.
- Accurate cross-platform mapping from port → PID/user/cmd is the hard part; requiring a perfect in-process implementation early is a scope trap.
- Auto “find next port” wrappers are inherently racy unless you can pass an already-bound socket to the child; treat wrappers as best-effort convenience, not a guarantee.
- `kill` needs strict, predictable safety semantics (own-user by default; escalation flags; TERM→KILL timeouts; clear output).

## Lower-level primitives (small, composable)
1. `probe(port, proto, v4/v6) -> free|in_use`: implemented via `bind()`/`listen()` (the only reliable “available?” check).
2. `who(port) -> {pid,user,cmd,addr,proto}`: implemented by delegating to OS tooling first (`lsof` on macOS, `ss`/`lsof` on Linux); replace later only if needed.
3. `kill(pid, mode)`: ownership checks + signal strategy (TERM then KILL after timeout).
4. `pick(prefer, range) -> port`: repeated `probe` with simple policy.
5. `coordination lock`: `flock`-based lockfile so concurrent `freeport run` invocations don’t race each other (does not stop non-freeport processes, but prevents self-collisions).
6. `--json` output everywhere: makes the tool agent/automation friendly from day 1.

## Recommended build approach (tightly-scoped MVP)
### v0.1 (single binary, no daemon)
- Implement in Go with Cobra.
- Commands:
  - `freeport list [--json]`: show listening TCP ports (best-effort PID/user/cmd).
  - `freeport who <port> [--json]`: details for a specific port.
  - `freeport kill <port> [--signal TERM|KILL] [--timeout 2s] [--force]`: safe-by-default port freeing.
  - `freeport pick --prefer 3000 --range 3000-3999 [--json]`: pick a free port using `probe`.
  - `freeport run --env PORT --prefer 3000 --range 3000-3999 -- <cmd...>`: best-effort wrapper with lockfile coordination.

### v0.2+ (only if demanded by users)
- Consider a daemon only if you need durable coordination semantics (leases/reservations) that other tools depend on.
- Avoid UI/tray/menu-bar until the CLI proves adoption.

## Success criteria (MVP)
- Installs quickly; no background services required.
- `list/who/kill/pick/run` work on macOS out of the box; Linux best-effort with `ss`/`lsof` availability.
- Non-interactive, stable output + exit codes; `--json` for automation.

