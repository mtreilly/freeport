# Freeport: Real-World Example Walkthrough

## The Problem Today

### Scenario 1: Without Freeport

```bash
# Terminal 1
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> 
> ➜  Local:   http://localhost:3000/

# [Vite is now running happily on port 3000]

# Terminal 2 (you want to start another project)
$ pnpm dev
> VITE v5.0.0  ready in 234 ms

> ❌ Error: Port 3000 is already in use

# Now what? Your options:
```

---

## Your Options Without Freeport

### Option A: Find and Kill the Old Process

```bash
# Find what's using port 3000
$ lsof -i :3000
COMMAND   PID USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
node    12345 user   23u  IPv4 0x1234      0t0  TCP *:3000 (LISTEN)

# Kill it
$ kill 12345

# Try again
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> 
> ✅ Success (but had to type 3 commands)
```

**Pain**: You had to remember `lsof`, parse output, and type the PID.

---

### Option B: Use a Different Port

```bash
# Start on port 3001
$ pnpm dev --port 3001
> VITE v5.0.0  ready in 234 ms
> 
> ➜  Local:   http://localhost:3001/

# But now you have to remember: "This project is on 3001, not 3000"
```

**Pain**: You have to remember which port each project uses.

---

### Option C: Wait and Hope

```bash
# Stop your old Vite (Ctrl+C)
# Wait a few seconds for port to release
# Try again
$ pnpm dev
> VITE v5.0.0  ready in 234 ms

# Sometimes this works, sometimes you get "Error: Port 3000 is already in use"
# Because the port is in TIME_WAIT state (needs 60s to clear)
```

**Pain**: Unreliable, frustrating, wastes time.

---

## What Freeport Does (Phase 1: Passive Mode)

### Installation (One-time)

```bash
$ brew install freeport
# Or: cargo install freeport

# Done! Freeport auto-starts in the background
```

---

### Now Try the Same Scenario

```bash
# Terminal 1
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> 
> ➜  Local:   http://localhost:3000/

# [Freeport silently records: Port 3000, Service: Vite, PID: 12345]

# Terminal 2 (you want to start another project)
$ pnpm dev
> VITE v5.0.0  ready in 234 ms

> ❌ Error: Port 3000 is already in use

# Same error as before, BUT now you have better tools!
```

---

### Option A: See What's Running (Much Easier)

```bash
# List all ports (clean, formatted output)
$ freeport list

PORT    SERVICE    PID     STATUS      LAST_SEEN     USER
3000    Vite       12345   active      10:30 AM     alice
5000    Django     67890   active      9:15 AM      alice
8080    Java       54321   stale       8:00 AM      alice (crashed)

# Clear, readable output! Much better than lsof
```

---

### Option B: Quick Kill (One Command)

```bash
# Kill whatever is on port 3000 (no need to find PID)
$ freeport kill 3000

> Killing Vite process (PID 12345)...
> Port 3000 freed

# Try again
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> 
> ✅ Success (1 command vs 3 commands without Freeport)
```

**Benefit**: Much faster and easier than `lsof | kill`

---

### Option C: Auto-Cleanup of Zombie Ports

```bash
# Scenario: Your Vite crashed (unhandled exception)
# Port 3000 is now "zombie" (stuck in TIME_WAIT)

# Without Freeport:
$ pnpm dev
> ❌ Error: Port 3000 is already in use
# You have to wait 60 seconds or manually kill

# With Freeport:
# Freeport detects the process is dead every 30 seconds
# Automatically marks it as "stale" and removes it

$ freeport list
PORT    SERVICE    PID     STATUS      LAST_SEEN
3000    Vite       12345   stale       10:32 AM (crashed)

# Freeport automatically cleans up stale entries
# You just wait up to 30 seconds and try again

$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> ✅ Success (automatic cleanup!)
```

**Benefit**: No more waiting or manual cleanup

---

## What Freeport Does (Phase 2: Smart Wrapper)

### Setup (One-time, per project)

```bash
# Option A: Add to package.json
{
  "scripts": {
    "dev": "freeport run --port 3000 -- pnpm dev"
  }
}

# Option B: Use npx (no package.json change)
$ npx @freeport/run --port 3000 -- pnpm dev
```

---

### Now Try the Same Scenario (Auto-Magic!)

```bash
# Terminal 1
$ pnpm dev  # or: npx @freeport/run --port 3000 -- pnpm dev
> freeport: Port 3000 available
> VITE v5.0.0  ready in 234 ms
> 
> ➜  Local:   http://localhost:3000/

# Terminal 2 (you want to start another project)
$ pnpm dev  # or: npx @freeport/run --port 3000 -- pnpm dev

> freeport: Port 3000 occupied by Vite (PID 12345)
> freeport: Finding next free port...
> freeport: Using port 3001 instead of 3000
> VITE v5.0.0  ready in 234 ms
> 
> ➜  Local:   http://localhost:3001/

# ✅ Success! Second project started automatically on 3001
```

**What happened**:
1. Freeport checked port 3000 → occupied
2. Found next free port (3001)
3. Started Vite on 3001 automatically
4. You don't have to do anything!

---

## Comparison: Side by Side

### Scenario: Starting two projects on port 3000

| Step | Without Freeport | With Freeport (Phase 1) | With Freeport (Phase 2) |
|------|------------------|-------------------------|-------------------------|
| Start project 1 | ✅ Success | ✅ Success | ✅ Success |
| Start project 2 | ❌ Error "already in use" | ❌ Error "already in use" | ✅ Auto-starts on 3001 |
| Your next step? | Manual kill or change port | `freeport kill 3000` (1 cmd) | Nothing! (already running) |
| Total time | 30-60 seconds | 5-10 seconds | 0 seconds |

---

## Complete Workflow Example

### Phase 1 Workflow (Passive Mode)

```bash
# Morning: Start working on project A
$ cd ~/projects/project-a
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> ➜  Local:   http://localhost:3000/

# Later: Switch to project B
$ cd ~/projects/project-b
$ pnpm dev
> ❌ Error: Port 3000 is already in use

# What do I do now?

# Option 1: See what's running
$ freeport list
PORT    SERVICE    PID     STATUS
3000    Vite       12345   active

# Option 2: Quick kill (fastest)
$ freeport kill 3000
> Killing Vite process (PID 12345)...

# Try again
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> ✅ Success

# Total time: ~5 seconds (1 command + 1 retry)
```

---

### Phase 2 Workflow (Smart Wrapper)

```bash
# Morning: Start working on project A
$ cd ~/projects/project-a
$ pnpm dev  # or: npx @freeport/run --port 3000 -- pnpm dev
> freeport: Port 3000 available
> VITE v5.0.0  ready in 234 ms
> ➜  Local:   http://localhost:3000/

# Later: Switch to project B
$ cd ~/projects/project-b
$ pnpm dev  # or: npx @freeport/run --port 3000 -- pnpm dev

> freeport: Port 3000 occupied by Vite (PID 12345)
> freeport: Using port 3001 instead of 3000
> VITE v5.0.0  ready in 234 ms
> ➜  Local:   http://localhost:3001/

# Total time: 0 seconds! (everything automatic)
```

---

## Advanced Scenarios

### Scenario 1: Working on Multiple Projects

```bash
# Project A (Vite)
$ cd ~/projects/a && pnpm dev
> freeport: Port 3000 available
> VITE v5.0.0  ready in 234 ms
> ➜  Local:   http://localhost:3000/

# Project B (Next.js)
$ cd ~/projects/b && npm run dev
> freeport: Port 3000 occupied, using 3001
> Next.js ready in 345 ms
> ➜  Local:   http://localhost:3001/

# Project C (Django)
$ cd ~/projects/c && python manage.py runserver
> freeport: Port 3000 occupied, using 3002
> Starting development server at http://127.0.0.1:3002/

# See what's running:
$ freeport list
PORT    SERVICE    PID     STATUS
3000    Vite       12345   active
3001    Next.js    23456   active
3002    Django     34567   active

# Kill all at once:
$ freeport kill-all --pattern "dev"
> Killing Vite (PID 12345)...
> Killing Next.js (PID 23456)...
> Killing Django (PID 34567)...
> All dev servers stopped
```

---

### Scenario 2: Zombie Cleanup

```bash
# Start Vite
$ pnpm dev
> VITE v5.0.0  ready in 234 ms

# [Vite crashes with unhandled exception]

# Try to restart immediately
$ pnpm dev
> ❌ Error: Port 3000 is already in use

# Without Freeport: You'd have to wait 60 seconds or find/kill manually

# With Freeport:
# Wait up to 30 seconds (cleanup interval)
$ freeport list
PORT    SERVICE    PID     STATUS      LAST_SEEN
3000    Vite       12345   stale       10:32 AM (crashed)

# Wait a few more seconds...
$ freeport list
PORT    SERVICE    PID     STATUS
# Port 3000 is gone! (automatically cleaned up)

# Try again
$ pnpm dev
> VITE v5.0.0  ready in 234 ms
> ✅ Success (automatic cleanup)
```

---

### Scenario 3: CI/CD Pipeline

```bash
# GitHub Actions workflow (same for GitLab, etc.)
name: Run Tests
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Freeport
        run: cargo install freeport
      
      - name: Reserve test port
        run: freeport reserve --pool test-ports --job-id $GITHUB_JOB_ID
      
      - name: Start services
        run: pnpm dev &
        env:
          PORT: 3000  # Freeport manages allocation
      
      - name: Run tests
        run: pnpm test
      
      - name: Release port
        if: always()
        run: freeport release --job-id $GITHUB_JOB_ID

# Benefits:
# - Multiple CI jobs can run on same port
# - Freeport automatically assigns different ports
# - No "port already in use" errors in CI
```

---

## Real User Stories

### Developer Alice (Frontend Dev)

**Problem**: "I work on 5 different React projects, all want port 3000"

**Without Freeport**:
```bash
# Every time I switch projects
$ cd project-a && pnpm dev
> Error: Port 3000 is already in use

# I have to:
# 1. Remember which project is running
# 2. Kill it or change port
# 3. Hope I remember which port I used
```

**With Freeport (Phase 2)**:
```bash
# Just run pnpm dev on every project
$ cd project-a && pnpm dev  # Port 3000
$ cd project-b && pnpm dev  # Port 3001 (auto)
$ cd project-c && pnpm dev  # Port 3002 (auto)

# Everything just works!
```

---

### Developer Bob (Full Stack)

**Problem**: "I run Django (port 8000), React (port 3000), and Postgres (port 5432) every morning"

**Without Freeport**:
```bash
# My morning script (sometimes fails)
#!/bin/bash
pnpm dev &
python manage.py runserver &
docker-compose up

# If I forget to kill previous day's processes:
# "Port 3000 already in use"
# "Port 8000 already in use"
# I have to manually kill them
```

**With Freeport**:
```bash
# My morning script (always works)
#!/bin/bash
freeport run --port 3000 -- pnpm dev &
freeport run --port 8000 -- python manage.py runserver &
docker-compose up

# Freeport handles conflicts automatically
# If a process is still running from yesterday:
# - Django: Auto-kills and restarts on 8000
# - React: Auto-uses next free port (3001, 3002, etc.)
# - Postgres: Same thing
```

---

### Team Shared Machine

**Problem**: "Alice and Bob share a dev server, both want to run their own apps on port 3000"

**Without Freeport**:
```bash
# Alice: Starts her app on port 3000
$ pnpm dev  # Port 3000

# Bob: Tries to start his app
$ pnpm dev
> Error: Port 3000 is already in use

# Bob has to:
# 1. Ask Alice to kill her process
# 2. Or use port 3001
# 3. Remember his app is on 3001
```

**With Freeport**:
```bash
# Alice: Starts her app
$ pnpm dev  # Port 3000

# Bob: Starts his app
$ pnpm dev
> freeport: Port 3000 occupied by alice@Vite (PID 12345)
> freeport: Using port 3001 instead of 3000
> VITE v5.0.0  ready in 234 ms
> ➜  Local:   http://localhost:3001/

# Both apps running, no coordination needed!

# Alice can check what's running:
$ freeport list
PORT    SERVICE    USER   PID
3000    Vite       alice  12345
3001    Vite       bob    23456

# Alice sees Bob's process, knows not to kill it
```

---

## Summary: What You Actually Do

### Starting pnpm dev when port 3000 is occupied

| Tool | What you type | What happens | Time taken |
|------|--------------|-------------|------------|
| **Nothing** | `pnpm dev` | ❌ Error "already in use" | 2 seconds |
| **Manual** | `lsof -i :3000`<br>`kill <PID>`<br>`pnpm dev` | ✅ Success | 30-60 seconds |
| **Freeport Phase 1** | `freeport kill 3000`<br>`pnpm dev` | ✅ Success | 5-10 seconds |
| **Freeport Phase 2** | `pnpm dev` (with wrapper) | ✅ Success (auto) | 0 seconds |

---

## Key Takeaways

### Phase 1 (Passive Mode)
- **Zero adoption**: Install and forget
- **Better visibility**: `freeport list` vs `lsof`
- **Easier kill**: `freeport kill 3000` vs `lsof | kill`
- **Auto cleanup**: Zombie ports removed automatically
- **Time saved**: 5-10 seconds vs 30-60 seconds

### Phase 2 (Smart Wrapper)
- **Auto conflict resolution**: No manual intervention
- **Zero configuration**: One npx command
- **Works everywhere**: Universal wrapper for any app
- **Time saved**: 0 seconds vs 30-60 seconds

### The Difference
- **Without Freeport**: Port conflicts cost you 30-60 seconds every time
- **With Freeport Phase 1**: Port conflicts cost 5-10 seconds
- **With Freeport Phase 2**: Port conflicts cost 0 seconds (automatic)

---

## Bottom Line

**You just run `pnpm dev` and it works.**

If port 3000 is occupied:
- **Phase 1**: `freeport kill 3000` and try again (5 seconds)
- **Phase 2**: Automatically uses next free port (0 seconds)

No more "Error: Port already in use" frustration! 🎉
