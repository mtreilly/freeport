# Freeport MVP Scope and Port Semantics

This document captures the MVP scope and the port semantics behind it.
Rationale is summarized in `REVIEW_CODEX.md`.

## MVP scope (no daemon)
- Single binary CLI.
- No background service, no SQLite registry, no IPC.
- Query the OS on demand for "what is listening right now".
- Provide small, composable primitives for humans and automation.

## What "busy" means
When Freeport says a port is "in use", it means a process is actively
LISTENing on that TCP port. This is the state that prevents another
server from binding to the same port.

## TIME_WAIT is not a listener
TIME_WAIT indicates a recently closed TCP connection; it does not hold a
LISTENing socket. We do not "clean up" TIME_WAIT. If a port is blocked,
there is almost always a LISTENing process still running (or an IPv4/IPv6
binding mismatch).

## Wrapper behavior is best-effort
`freeport run` chooses a port using bind probes plus a lockfile to avoid
collisions between concurrent Freeport invocations. It cannot prevent
races with non-Freeport processes, so it is best-effort and should never
claim to guarantee exclusivity.

## Kill safety defaults
- Refuse to kill processes owned by other users unless `--force`.
- Default signal is TERM; optional escalation to KILL after a timeout.
- Deterministic output and exit codes (no prompts).

