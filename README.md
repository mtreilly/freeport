# Freeport

Freeport is a small, no-daemon CLI for answering "what is on this port?"
and freeing a port quickly. It is designed for local dev workflows and
automation.

## Scope (v0.1)
- Single binary CLI.
- No background service or registry.
- Best-effort wrappers, deterministic output.

For the design rationale and port semantics, see `MVP_SCOPE.md`.

## Build
```bash
go build ./...
```

## Usage

### List listening TCP ports
```bash
./freeport list
```

JSON output:
```bash
./freeport list --json
```

### See who is on a port
```bash
./freeport who 3000
```

JSON output:
```bash
./freeport who 3000 --json
```

### Kill listeners on a port (safe defaults)
```bash
./freeport kill 3000
```

Options:
```bash
./freeport kill 3000 --signal INT --timeout 1s
./freeport kill 3000 --force
./freeport kill 3000 --json
```

### Pick a free port
```bash
./freeport pick --prefer 3000 --range 3000-3999
```

JSON output:
```bash
./freeport pick --prefer 3000 --range 3000-3999 --json
```

Let the OS pick (ephemeral):
```bash
./freeport pick --prefer 0
```

### Check a port (automation-friendly exit codes)
```bash
./freeport check 3000
```

JSON output:
```bash
./freeport check 3000 --json
```

### Run a command with a chosen PORT (best-effort)
```bash
./freeport run --prefer 3000 --range 3000-3999 -- env | rg '^PORT='
```

## Notes
- `list` and `who` use `lsof` on macOS and `ss` on Linux when available.
- `run` is best-effort and cannot prevent races with non-Freeport processes.
- On Linux, `ss` may omit PID/command without sufficient permissions, so
  those fields can be blank.
