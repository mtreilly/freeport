# fp

`fp` (freeport) is a small, no-daemon CLI for answering "what is on this port?"
and freeing a port quickly. Designed for local dev workflows and automation.

## Install

```bash
./install.sh
# or
go install .
```

## Quick Start

```bash
fp list              # list all listening ports
fp list node         # filter by process name
fp who 3000          # detailed info on port 3000
fp kill 3000         # kill process on port 3000
fp doctor            # check system dependencies
```

## Usage

### List listening TCP ports
```bash
fp list                      # all ports
fp list node                 # filter by command name
fp list --port 3000          # filter by port
fp list --cwd .              # listeners in this project and subdirectories
fp list --unique             # dedupe by port+PID
fp list -v                   # show full executable path
fp list --json               # JSON output
```

`--cwd` resolves symlinks and matches process working directories, not executable
locations. Listeners whose working directory cannot be read are excluded.
It can be combined with the process-name and port filters.

### See who is on a port
```bash
fp who 3000
fp who 3000 --json
```

### Kill listeners on a port
```bash
fp kill 3000                          # SIGTERM with 2s timeout
fp kill 3000 --signal INT --timeout 1s
fp kill 3000 --force                  # override user check
fp kill 3000 --dry-run                # preview targets
fp kill 3000 --json                   # JSON result on stdout; progress on stderr
```

### Pick a free port
```bash
fp pick                               # default: prefer 3000
fp pick --prefer 8080 --range 8000-8999
fp pick --prefer 0                    # OS-assigned ephemeral
```

### Check a port
```bash
fp check 3000                # exit 0=free, 1=in-use, 2=error
fp check 3000 --wait 5s      # wait up to 5s for port to free
fp check 3000 -q             # no normal output; errors still go to stderr
```

`--quiet` (`-q`) and `--json` are mutually exclusive.

### Run a command with PORT env var
```bash
fp run -- node server.js
fp run --prefer 8080 -- python app.py
fp run --env API_PORT -- ./myserver
```

`run` preserves the child command's exit code. If the child is terminated by a
signal, it exits with 128 + the signal number (for example, 143 for SIGTERM).
Child output is passed through; the chosen-port message goes to stderr.

### Shell completion
```bash
# Bash
eval "$(fp completion bash)"

# Zsh
eval "$(fp completion zsh)"

# Fish
fp completion fish | source
```

With completion loaded, `fp who <Tab>` and `fp kill <Tab>` suggest currently
listening TCP ports with process names and PIDs when available. Suggestions
are queried on demand, deduplicated, and filtered by any port prefix you type.

### System check
```bash
fp doctor
```

## Notes
- Uses `lsof` on macOS and `ss` on Linux
- `run` is best-effort; cannot prevent races with non-fp processes
- On Linux, `ss` may omit PID/command without root

## FAQ

**Why no daemon?**
All data comes from the OS at query time. No background services needed.

**Why not clean up TIME_WAIT?**
TIME_WAIT is a closed connection, not a listener. If a port is blocked,
there's almost always a live listener.

**Does `run` guarantee exclusivity?**
No. It uses lockfiles to avoid collisions between fp invocations, but
external processes can still race.
