# Freeport: Port Management Tool - Rough Ideas & Planning Document

## Project Overview

**Goal**: Build a lightweight, cross-platform port management system for Linux and macOS that allows services to register port usage and enables other applications to query available ports.

**Core Philosophy**: Small memory footprint, unprivileged operation, automatic cleanup, developer-friendly.

**Platforms**: macOS (menu bar app in Swift) + Linux (daemon in Rust)

---

## Table of Contents
1. [Core Problems Solved](#core-problems-solved)
2. [High-Level Architecture](#high-level-architecture)
3. [Platform-Specific Implementation](#platform-specific-implementation)
4. [Key Technical Decisions](#key-technical-decisions)
5. [Database Schema](#database-schema)
6. [Open Questions & Design Tradeoffs](#open-questions--design-tradeoffs)
7. [Implementation Phases](#implementation-phases)
8. [Security Considerations](#security-considerations)
9. [Performance & Memory Optimization](#performance--memory-optimization)
10. [Testing Strategy](#testing-strategy)
11. [Existing Solutions & Inspiration](#existing-solutions--inspiration)

---

## Core Problems Solved

### 1. Port Conflicts in Development
- **Problem**: Developers frequently encounter "port already in use" errors (3000, 8080, 8000, etc.)
- **Current Solutions**: Kill ports manually (lsof/kill), use different ports per project, Docker dynamic ports
- **Our Solution**: Centralized port registry with automatic conflict detection and resolution

### 2. No Visibility Into Port Usage
- **Problem**: Developers don't know what services are running on which ports without manual inspection
- **Current Solutions**: `lsof -i`, `netstat -tulpn`, activity monitors (macOS)
- **Our Solution**: Real-time UI showing all registered ports, service names, and metadata

### 3. Zombie Port Allocations
- **Problem**: Processes crash without releasing ports, leaving ports marked as "in use"
- **Current Solutions**: Manual cleanup, server restart, or system reboot in extreme cases
- **Our Solution**: Automatic heartbeat monitoring and cleanup of dead processes

### 4. Multi-Developer Conflicts
- **Problem**: Multiple developers on same machine conflict over common ports
- **Current Solutions**: Ad-hoc coordination, different port ranges per developer
- **Our Solution**: Per-user namespaces or developer-specified port pools

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Applications                      │
│  (Django, Vite, Java servers, custom apps, etc.)               │
└──────────────┬───────────────────────────┬──────────────────────┘
               │                           │
               ▼                           ▼
        ┌──────────────┐           ┌──────────────┐
        │   macOS CLI   │           │   Linux CLI   │
        │   (Swift)    │           │   (Rust)      │
        └──────┬───────┘           └──────┬───────┘
               │                           │
               │        IPC Layer          │
               ▼                           ▼
        ┌──────────────────────────────────────────────┐
        │     Daemon / Core Service (Shared Core)       │
        │  - SQLite Database (WAL mode)               │
        │  - Port Allocation Logic                     │
        │  - Cleanup & Monitoring                     │
        └──────────────┬─────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
        ▼                             ▼
┌───────────────────┐       ┌──────────────────┐
│  macOS GUI        │       │  Linux GUI/CLI   │
│  (Menu Bar App)   │       │  (System Tray/   │
│  - Status View     │       │   TUI/CLI)       │
│  - Quick Actions   │       │  - Port List     │
│  - Configuration   │       │  - CLI Commands  │
└───────────────────┘       └──────────────────┘
```

### Components

#### 1. Core Daemon (Platform-Agnostic Logic)
- **Language**: Rust (for Linux) / Swift (for macOS) - or shared protocol
- **Responsibilities**:
  - Manage SQLite database
  - Handle port registration/deregistration
  - Detect system port usage (cross-platform abstraction)
  - Monitor registered services (heartbeats, PID checks)
  - Provide IPC interface

#### 2. Client Libraries (Optional)
- Swift library for macOS apps
- Rust library for Linux apps
- Optional bindings for other languages (Python, Node.js, etc.)
- Simple API: `register_port(port, service_name, metadata)`, `get_free_port(preferred_range)`

#### 3. User Interfaces

**macOS**:
- Menu bar application (NSStatusItem)
- Shows port count, conflicts, quick status
- Popover with full port list
- Quick actions: "Kill all non-registered on port X", "Reserve port X"

**Linux**:
- System tray indicator (libayatana-appindicator)
- Alternative: Terminal UI (tui-rs)
- CLI tool with subcommands: `freeport list`, `freeport claim 3000`, `freeport release 3000`

---

## Platform-Specific Implementation

### macOS Implementation (Swift)

#### Framework & Libraries
- **Core**: Foundation, AppKit (for NSStatusItem)
- **Database**: GRDB.swift or SQLite.swift
- **UI**: SwiftUI for popover/window views
- **IPC**: Unix Domain Sockets (via Foundation's Network framework)
- **System Integration**:
  - `lsof` via Process for port detection
  - `kill` for terminating processes
  - LaunchAgent for auto-start on boot
  - NSWorkspace for app lifecycle

#### Key Components

**1. Menu Bar Application**
```swift
@main
struct FreeportApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
}

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusItem: NSStatusItem?
    var popover: NSPopover?
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        // Create status item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem?.button?.image = NSImage(systemSymbolName: "network", accessibilityDescription: "Freeport")
        
        // Configure popover
        popover = NSPopover()
        popover?.contentViewController = NSHostingController(rootView: PortListView())
        popover?.behavior = .transient
        
        // Setup IPC server
        startIPCServer()
    }
}
```

**2. Port Detection (macOS)**
```swift
// Method 1: Using lsof (legacy but reliable)
func getProcessesOnPort(_ port: Int) async throws -> [PortInfo] {
    let output = try await Process.execute(
        "/usr/sbin/lsof",
        arguments: ["-i", ":\(port)", "-P", "-n", "-sTCP:LISTEN"]
    )
    // Parse lsof output
    return parseLsofOutput(output)
}

// Method 2: Using NWConnection (programmatic, more modern)
func isPortAvailable(_ port: Int) -> Bool {
    let endpoint = NWEndpoint.hostPort(host: .ipv4(.loopback), port: .init(integerLiteral: UInt16(port)))
    let connection = NWConnection(to: endpoint, using: .tcp)
    // Try to bind
}
```

**3. Database Integration (GRDB)**
```swift
import GRDB

struct PortRegistration: Codable, FetchableRecord, PersistableRecord {
    var id: Int64
    var port: Int
    var serviceName: String
    var pid: Int?
    var metadata: String // JSON
    var createdAt: Date
    var lastHeartbeat: Date
    var status: String // "active", "claimed", "reserved"
}

let dbQueue = try DatabaseQueue(path: "/Users/Shared/freeport.db")
try dbQueue.write { db in
    try PortRegistration(port: 3000, serviceName: "Django Dev Server", pid: 12345, ...).insert(db)
}
```

#### Build & Distribution
- **Xcode Project**: macOS App target
- **Signed .app**: For distribution outside App Store
- **System Requirements**: macOS 13.0+ (modern APIs)
- **Permissions**: Full Disk Access (to run lsof/kill without password prompts)

---

### Linux Implementation (Rust)

#### Key Crates
```toml
[dependencies]
rusqlite = { version = "0.30", features = ["bundled"] }
tokio = { version = "1", features = ["full"] }
tokio-util = { version = "0.7", features = ["net"] }
nix = "0.27" # For low-level system calls
sysinfo = "0.30" # For process monitoring
clap = { version = "4", features = ["derive"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"

# Optional for GUI
libayatana-appindicator = "0.1" # System tray
crossterm = "0.27" # Terminal UI
tui = "0.19" # Alternative TUI
```

#### Key Components

**1. Port Detection (Linux)**
```rust
use nix::sys::socket::{socket, bind, sockopt, getsockname, AddressFamily, SockType, SockFlag, SockProtocol};
use nix::sys::socket::{SockaddrIn, InetAddr};
use std::os::unix::io::AsRawFd;

// Method 1: Try to bind (most reliable)
fn is_port_available(port: u16) -> bool {
    let fd = match socket(
        AddressFamily::Inet,
        SockType::Stream,
        SockFlag::empty(),
        None
    ) {
        Ok(fd) => fd,
        Err(_) => return false,
    };
    
    let addr = SockaddrIn::new(127, 0, 0, 1, port);
    match bind(fd.as_raw_fd(), &addr) {
        Ok(_) => true,
        Err(_) => false,
    }
}

// Method 2: Read from /proc/net/tcp (faster for bulk checks)
fn get_listening_ports() -> HashSet<u16> {
    let mut ports = HashSet::new();
    if let Ok(content) = std::fs::read_to_string("/proc/net/tcp") {
        for line in content.lines().skip(1) { // Skip header
            if let Some(port_hex) = line.split_whitespace().nth(1) {
                if let Ok(port) = u16::from_str_radix(port_hex.split(':').last().unwrap_or("0"), 16) {
                    // Check if in LISTEN state (0A)
                    if line.contains("0A") {
                        ports.insert(port);
                    }
                }
            }
        }
    }
    ports
}

// Method 3: Parse /proc/[PID]/fd to map ports to processes
fn get_process_on_port(port: u16) -> Option<u32> {
    // Walk /proc/*/fd to find socket inodes
    // Match against /proc/net/tcp
    // Return PID
}
```

**2. Database Operations (rusqlite)**
```rust
use rusqlite::{Connection, params, Result};

#[derive(Debug)]
struct PortRegistration {
    id: i64,
    port: i32,
    service_name: String,
    pid: Option<i32>,
    metadata: String,
    created_at: i64,
    last_heartbeat: i64,
    status: String,
}

fn register_port(conn: &mut Connection, reg: &PortRegistration) -> Result<()> {
    conn.execute(
        "INSERT INTO ports (port, service_name, pid, metadata, created_at, last_heartbeat, status)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
        params![reg.port, reg.service_name, reg.pid, reg.metadata, 
                reg.created_at, reg.last_heartbeat, reg.status],
    )?;
    Ok(())
}
```

**3. IPC Server (Unix Domain Socket)**
```rust
use tokio::net::UnixListener;
use tokio::io::{AsyncBufReadExt, BufReader};

async fn run_ipc_server(socket_path: &str) -> Result<()> {
    let listener = UnixListener::bind(socket_path)?;
    
    loop {
        let (mut stream, _) = listener.accept().await?;
        tokio::spawn(async move {
            let mut reader = BufReader::new(&stream);
            let mut line = String::new();
            while reader.read_line(&mut line).await.unwrap() > 0 {
                // Process command: "REGISTER 3000 \"Django\""
                let response = handle_command(&line);
                stream.write_all(response.as_bytes()).await.unwrap();
                line.clear();
            }
        });
    }
}
```

**4. System Tray (Linux)**
```rust
// Using libayatana-appindicator bindings (via FFI or wrapper)
// Alternative: Use just CLI + TUI

// TUI with tui-rs
fn run_tui(db: &Database) -> Result<()> {
    let terminal = ratatui::init();
    // Show port list, allow keyboard navigation
}
```

#### systemd Integration
```ini
# /etc/systemd/system/freeport.service
[Unit]
Description=Freeport Port Management Service
After=network.target

[Service]
Type=simple
User=freeport
ExecStart=/usr/local/bin/freeportd
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

---

## Key Technical Decisions

### 1. Database Choice: SQLite (with WAL mode)

**Why SQLite?**
- Zero configuration (no separate database server)
- Excellent for read-heavy workloads
- WAL mode enables concurrent readers
- Single file = easy backup/restore
- Built into OS on both platforms

**SQLite Configuration:**
```sql
PRAGMA journal_mode = WAL;           -- Enable Write-Ahead Logging
PRAGMA synchronous = NORMAL;          -- Faster than FULL, safe enough
PRAGMA cache_size = -2000;            -- 2MB cache
PRAGMA temp_store = MEMORY;           -- Keep temp tables in RAM
PRAGMA mmap_size = 268435456;        -- 256MB memory-mapped I/O
PRAGMA page_size = 4096;              -- Match filesystem block size
```

**Concurrency Strategy:**
- Multiple readers (queries, port checks)
- Single writer (registrations, updates)
- Clients use WAL for non-blocking reads
- Writer queues transactions with busy timeout

### 2. IPC Mechanism: Unix Domain Sockets

**Why UDS over HTTP?**
- 10-20x faster (0.01-0.05ms vs 0.1-1ms latency)
- Lower overhead (no TCP/IP stack)
- File-system permissions for security
- No risk of accidental network exposure

**Protocol Design:**
```
Simple request-response protocol:
- Text-based over UDS
- JSON or simple key-value format
- Commands: REGISTER, CLAIM, RELEASE, LIST, STATUS, PING

Example:
Client → Server: {"cmd": "REGISTER", "port": 3000, "service": "Django"}
Server → Client: {"status": "success", "port": 3000, "expires": "1h"}
```

**Socket Location:**
- macOS: `/var/run/freeport.sock`
- Linux: `/var/run/freeport.sock`
- Permissions: 0770 (owner + group read/write)

### 3. Port Detection Strategy

**Hybrid Approach:**
1. **Fast check**: Query internal database (O(1))
2. **Verification**: Try to bind to port (fast, reliable)
3. **Cross-check**: Scan system processes (slower, thorough)

**When to use which:**
- Database check: Every query (instant)
- Bind attempt: Before registration (<1ms)
- Full scan: Periodic cleanup (every 30-60s)

### 4. Service Lifecycle Management

**Registration Flow:**
```
1. Client requests port (specific or "next available")
2. Daemon checks database + system
3. If port free:
   - Create registration entry
   - Return success with lease duration
4. Client sends heartbeats (every 30-60s)
5. If heartbeat missed (2x interval), mark as stale
6. Cleanup daemon removes stale entries
```

**Heartbeat Mechanism:**
- Optional: Some clients can just register once
- Required: Long-running services should heartbeat
- Tolerance: Allow 1-2 missed heartbeats before cleanup

---

## Database Schema

```sql
-- Ports table (main registry)
CREATE TABLE ports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    port INTEGER NOT NULL UNIQUE,
    service_name TEXT NOT NULL,
    pid INTEGER,                    -- NULL for reserved ports
    metadata TEXT,                  -- JSON: {"user": "alice", "project": "myapp"}
    status TEXT NOT NULL,           -- 'active', 'claimed', 'reserved', 'stale'
    created_at INTEGER NOT NULL,     -- Unix timestamp
    expires_at INTEGER,              -- NULL for indefinite
    last_heartbeat INTEGER,
    CONSTRAINT port_range CHECK (port >= 1 AND port <= 65535),
    CONSTRAINT valid_status CHECK (status IN ('active', 'claimed', 'reserved', 'stale'))
);

-- Indexes for performance
CREATE INDEX idx_port_status ON ports(status);
CREATE INDEX idx_port_expires ON ports(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_port_service ON ports(service_name);

-- Port pools/aliases (optional)
CREATE TABLE port_pools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    ports TEXT NOT NULL,            -- JSON array: [3000, 3001, 3002]
    users TEXT,                     -- JSON array of user IDs with access
    created_at INTEGER NOT NULL
);

-- Audit log (for debugging)
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    action TEXT NOT NULL,            -- 'register', 'release', 'heartbeat', 'cleanup'
    port INTEGER,
    service_name TEXT,
    pid INTEGER,
    details TEXT                     -- JSON
);

-- Configuration
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Insert default config
INSERT INTO config (key, value) VALUES 
    ('default_lease_duration', '3600'),  -- 1 hour
    ('heartbeat_interval', '30'),
    ('stale_threshold', '120'),
    ('cleanup_interval', '60'),
    ('preferred_ports', '3000,3001,3002,8000,8001,8080,8081,5000,5001');
```

---

## Open Questions & Design Tradeoffs

### 1. Single Daemon vs. Per-User Daemons

**Option A: Single System Daemon**
- Pros: Single source of truth, simpler coordination
- Cons: Requires root or special permissions, multi-user conflicts

**Option B: Per-User Daemons**
- Pros: No root required, isolated per user
- Cons: Can't prevent inter-user conflicts, multiple instances

**Recommendation**: Per-user daemons with optional system daemon for coordination

### 2. Centralized Registry vs. Distributed Locking

**Option A: Centralized Database**
- Pros: Simple, consistent, easy to query
- Cons: Single point of failure, requires daemon always running

**Option B: File-Based Locking**
- Pros: Works without daemon, resilient
- Cons: Slower, harder to query, potential for stale locks

**Recommendation**: Centralized database with fallback to file locking

### 3. Automatic Port Selection Algorithm

**Option A: Try Bind Approach**
- Algorithm: Try port X, if EADDRINUSE, try X+1
- Pros: Simple, guarantees free port
- Cons: Can be slow if many ports in use

**Option B: Query OS First**
- Algorithm: Query netstat/lsof, find first free port
- Pros: Fast, avoids blind attempts
- Cons: Still race condition risk

**Option C: Hybrid**
- Algorithm: Preferred ports list, then range search, then OS-assigned (port 0)
- Pros: Best of both worlds
- Cons: More complex

**Recommendation**: Hybrid with configurable preference lists

### 4. Port Reservation vs. Port Usage Tracking

**Option A: Just Track (Passive)**
- Daemon observes what's running, reports to user
- Pros: Non-invasive, always accurate
- Cons: Can't prevent conflicts, only report them

**Option B: Reservation System (Active)**
- Applications must register before using ports
- Pros: Can prevent conflicts, allocate intelligently
- Cons: Requires client adoption, can break existing apps

**Recommendation**: Both modes coexist
- Track passively (always running)
- Reservation mode optional (for cooperating apps)

### 5. Memory Footprint vs. Feature Richness

**Constraint**: Goal is small memory (<50MB)

**Minimal Approach:**
- SQLite with minimal cache
- No GUI, CLI only
- Simple IPC
- Periodic scans

**Rich Approach:**
- Larger SQLite cache
- Full GUI (macOS menu bar + Linux tray)
- Real-time monitoring (inotify/fsevents)
- Advanced features (port pools, analytics)

**Recommendation**: Start minimal, add features progressively
- Core daemon: <10MB
- GUI app: <30MB (macOS), <20MB (Linux)
- Total: <50MB

### 6. Security Model

**Question**: How to prevent unauthorized port registration/killing?

**Option A: File System Permissions**
- Socket file permissions (0770)
- Database file permissions
- Pros: Simple, works with Unix permissions
- Cons: Can't differentiate within same user

**Option B: Token-Based Auth**
- Clients get auth token on first registration
- Required for release/kill operations
- Pros: More fine-grained control
- Cons: More complex, token management

**Option C: PID Ownership**
- Only process owner can release its port
- Root can override
- Pros: Simple, intuitive
- Cons: PID reuse attacks (rare)

**Recommendation**: Start with file permissions + PID ownership, add token auth if needed

### 7. Cross-Platform Code Sharing

**Challenge**: Swift (macOS) vs. Rust (Linux) = different languages

**Option A: Separate Implementations**
- Platform-specific code only
- Share protocol (IPC, DB schema)
- Pros: Use best language for each platform
- Cons: Duplicated business logic

**Option B: Shared Core Language**
- Write core in Rust, call from Swift via FFI
- Pros: Single implementation
- Cons: Swift FFI overhead, less "Swifty" macOS code

**Option C: Protocol-Only Sharing**
- Define protocol (SQLite + IPC)
- Implement separately per platform
- Pros: Clean separation
- Cons: Risk of divergence

**Recommendation**: Option A (separate implementations, shared protocol)
- Document protocol clearly
- Shared test suite to verify compatibility
- Eventually consider core Rust library if complexity grows

### 8. Crash Recovery & Data Consistency

**Scenario**: Daemon crashes mid-transaction

**Approach**:
- SQLite handles atomic transactions
- WAL mode ensures no corruption
- On restart:
  1. Scan for inconsistent entries
  2. Verify against actual system state
  3. Clean up stale registrations
  4. Restore from last known good state

**Recovery Procedures**:
```bash
# Checkpoint WAL file if needed
sqlite3 freeport.db "PRAGMA wal_checkpoint(TRUNCATE);"

# Validate database integrity
sqlite3 freeport.db "PRAGMA integrity_check;"

# Restore from backup if corrupted
cp freeport.db.backup freeport.db
```

### 9. Deployment & Distribution

**macOS**:
- Signed .app bundle
- Install to `/Applications`
- Auto-start via LaunchAgent
- Update: Sparkle or manual

**Linux**:
- System package: .deb, .rpm, Arch AUR
- Binary release (statically linked)
- systemd service file
- Package manager integration?

**CI/CD**:
- GitHub Actions
- macOS: Build on macOS runner, sign, notarize
- Linux: Build on multiple distros, create packages
- Automated tests on both platforms

---

## Implementation Phases

### Phase 1: Core MVP (Weeks 1-3)

**Goal**: Basic port registration and querying

**Features**:
- [ ] SQLite database with schema
- [ ] CLI tool: `freeport register 3000 "Django"`
- [ ] CLI tool: `freeport list`
- [ ] Basic port detection (try bind)
- [ ] Single-platform focus (pick one)
- [ ] No GUI, no IPC server yet

**Tech Choices**:
- Start with Rust (easier to prototype)
- SQLite via rusqlite
- CLI via clap

**Deliverables**:
- Working binary that can register/list ports
- Basic tests
- Documentation

### Phase 2: IPC Server (Weeks 4-5)

**Goal**: Enable client-server communication

**Features**:
- [ ] Unix domain socket server
- [ ] Request protocol definition
- [ ] Client library (first in same language)
- [ ] Handle multiple concurrent connections
- [ ] Error handling and timeouts

**Protocol**:
```
REGISTER port service_name [pid] [metadata]
RELEASE port
HEARTBEAT port
LIST [status]
STATUS port
```

**Deliverables**:
- Working IPC server
- Client library examples
- Protocol documentation

### Phase 3: Platform-Specific GUIs (Weeks 6-8)

**macOS**:
- [ ] Menu bar application (Swift)
- [ ] Port list view
- [ ] Click to view details
- [ ] Quick actions (kill, reserve)

**Linux**:
- [ ] System tray indicator
- [ ] Or: TUI with tui-rs
- [ ] Same functionality as macOS

**Deliverables**:
- Installable .app (macOS)
- TUI binary (Linux)
- Integration tests

### Phase 4: Cross-Platform (Weeks 9-10)

**Goal**: Full macOS + Linux support

**Tasks**:
- [ ] Port code to Swift (macOS daemon)
- [ ] Ensure IPC protocol compatibility
- [ ] Test cross-platform scenarios
- [ ] Build and packaging for both

**Deliverables**:
- macOS package
- Linux packages (.deb, .rpm)
- Unified documentation

### Phase 5: Advanced Features (Weeks 11-12)

**Features**:
- [ ] Port pools/groups
- [ ] Per-user namespaces
- [ ] Automatic cleanup daemon
- [ ] Conflict resolution strategies
- [ ] Metrics and analytics

**Deliverables**:
- Advanced CLI commands
- UI enhancements
- Performance improvements

### Phase 6: Polish & Release (Weeks 13-14)

**Tasks**:
- [ ] Comprehensive testing
- [ ] Performance optimization
- [ ] Security audit
- [ ] Documentation (user guide, API docs)
- [ ] Website/GitHub setup
- [ ] v1.0 release

**Deliverables**:
- Stable v1.0 release
- Complete documentation
- CI/CD pipeline
- Website

---

## Security Considerations

### 1. Privilege Separation

**Principle**: Run with minimal necessary privileges

**Approach**:
- Daemon runs as unprivileged user (`freeport`)
- No root access required
- Users can only manage their own ports
- Optional system daemon for coordination (requires root)

### 2. File Permissions

**Database File**:
```
/var/lib/freeport/freeport.db    -rw-r-----  freeport:freeport
/var/run/freeport.sock            srw-rw----  freeport:freeport
```

**Configuration**:
```
/etc/freeport/config.yaml         -rw-r--r--  root:freeport
/home/user/.freeport/config.yaml   -rw-------  user:user
```

### 3. IPC Security

**Socket Permissions**:
- Owner: `freeport` user
- Group: `freeport` group (add users who need access)
- Mode: 0770 (owner + group only)

**Authentication** (optional):
- Simple token on first connection
- Or rely on file permissions

### 4. Port Spoofing Prevention

**Attack Vector**: Malicious process claims port for legit service

**Mitigations**:
- PID verification: Check if PID actually owns the port
- Heartbeat requirement: Process must prove it's running
- User isolation: Users can only manage their own processes
- Optional: Cryptographic signing of registration

### 5. Denial of Service Prevention

**Attack Vectors**:
- Register all ports (exhaust pool)
- Rapid requests (overwhelm daemon)

**Mitigations**:
- Per-user port limits
- Rate limiting on requests
- Expire stale registrations quickly
- CAPTCHA for unauthenticated requests (overkill?)

### 6. Data Integrity

**Database Protection**:
- SQLite's atomic transactions
- WAL mode prevents corruption
- Regular backups
- File system permissions

**Backup Strategy**:
```bash
# Automated daily backup
0 2 * * * cp /var/lib/freeport/freeport.db /var/backups/freeport-$(date +%Y%m%d).db
```

### 7. Audit Logging

**Track**:
- All registration/release actions
- Failed attempts (potential attacks)
- PID/port mismatches
- Unusual patterns (rapid port claiming)

**Review**:
- Regular log rotation
- Alert on suspicious activity
- Optional: Forward to syslog

---

## Performance & Memory Optimization

### 1. Memory Footprint Targets

**Components**:
- Core daemon: <10MB
- macOS GUI: <30MB
- Linux GUI/TUI: <20MB
- Total per system: <50MB

**Strategies**:
- Lazy loading (load views only when shown)
- Limited SQLite cache (2MB)
- Connection pooling (max 5 connections)
- Efficient data structures (hash maps, not lists)

### 2. Database Optimization

**SQLite Settings**:
```sql
PRAGMA cache_size = -2000;           -- 2MB cache
PRAGMA temp_store = MEMORY;           -- Keep temp tables in RAM
PRAGMA mmap_size = 268435456;        -- 256MB memory-mapped I/O
PRAGMA page_size = 4096;              -- Match filesystem block size
PRAGMA journal_mode = WAL;            -- Write-Ahead Logging
PRAGMA synchronous = NORMAL;           -- Faster than FULL
```

**Query Optimization**:
- Create appropriate indexes
- Use prepared statements
- Batch inserts/updates
- Avoid SELECT *
- Use LIMIT for large result sets

### 3. IPC Performance

**Optimizations**:
- Keep connection reuse
- Minimize serialization overhead
- Use binary protocol if JSON is too slow
- Connection pooling

**Benchmark Target**:
- <1ms average latency
- <5ms 99th percentile
- Handle 1000+ requests/second

### 4. Port Detection Optimization

**Strategy**:
- Cache system port state (refresh every 30s)
- Use /proc/net/tcp on Linux (fast bulk read)
- Prefer bind test over lsof subprocess
- Incremental updates (track changes, not full scan)

**Algorithm**:
```
1. Check database cache (O(1))
2. If port marked free, try bind (<1ms)
3. On periodic cleanup (every 30s):
   - Full system scan
   - Update cache
   - Clean stale entries
```

### 5. Monitoring & Profiling

**Tools**:
- macOS: Instruments, Activity Monitor
- Linux: perf, valgrind, heaptrack
- Custom metrics: response time, memory usage

**Key Metrics**:
- Memory RSS/VSZ
- Request latency (p50, p95, p99)
- Database query time
- Cleanup cycle duration

---

## Testing Strategy

### 1. Unit Tests

**Coverage Goals**:
- Database operations: 100%
- IPC protocol: 100%
- Port detection logic: 100%
- CLI commands: 90%

**Test Frameworks**:
- Rust: built-in `cargo test`, plus `criterion` for benchmarks
- Swift: XCTest, plus Quick/Nimble for BDD

**Example Test**:
```rust
#[test]
fn test_port_registration() {
    let db = setup_test_db();
    register_port(&db, 3000, "Test").unwrap();
    assert!(is_port_registered(&db, 3000));
    assert!(!is_port_registered(&db, 3001));
}
```

### 2. Integration Tests

**Scenarios**:
- Multi-process concurrent registration
- Client-server communication
- Cross-platform protocol compatibility
- Cleanup of dead processes

**Test Infrastructure**:
- Spin up real daemon process
- Connect multiple clients
- Verify database consistency
- Measure performance

### 3. E2E Tests

**User Workflows**:
1. Developer starts app, registers port 3000
2. Another developer tries to claim 3000 → fails
3. First app crashes, second developer retries → succeeds
4. List ports via CLI
5. Kill process, verify cleanup

**Automation**:
- Use real services (Python, Node, etc.)
- Simulate crashes
- Verify UI updates

### 4. Performance Tests

**Benchmarks**:
- Registration throughput (req/sec)
- Query latency (p50, p95, p99)
- Memory usage over time (leaks?)
- Cleanup cycle duration

**Tools**:
- Rust: criterion
- macOS: Instruments
- Linux: hyperfine, perf

### 5. Cross-Platform Tests

**Compatibility Matrix**:
- macOS 13, 14, 15
- Ubuntu 22.04, 24.04
- Fedora 38, 39
- Arch Linux (rolling)

**Test Strategy**:
- CI runs on all platforms
- Manual testing on real machines
- Beta testers for edge cases

### 6. Security Tests

**Scenarios**:
- Unauthorized port registration
- PID spoofing
- DoS attacks (rapid requests)
- Database injection attacks

**Tools**:
- AFL (fuzzing)
- Bandit/RustSec (static analysis)
- Manual penetration testing

---

## Existing Solutions & Inspiration

### Tools Analyzed

**1. KillPorts (macOS-only)**
- Focus: Kill stuck processes on common ports
- Inspiration: One-click conflict resolution
- Missing: Cross-platform, registration system

**2. Port Kill (multi-platform)**
- Focus: Find and free ports
- Features: Smart restart, guard mode
- Inspiration: Guard mode (auto-restart crashed services)
- Missing: Port registration, macOS menu bar

**3. Portless (macOS)**
- Focus: Tiny tool to kill occupied ports
- Inspiration: Simplicity, minimal footprint
- Missing: Linux support, prevention

**4. simple-service-registry (Go)**
- Focus: Service discovery
- Features: Self-registration, persistent storage
- Inspiration: Registration protocol
- Missing: Port management focus

**5. Kapeta local-cluster-service**
- Focus: Local service discovery
- Features: Port management, proxy
- Inspiration: Proxying for traffic inspection
- Missing: Menu bar UI

### What We're Doing Differently

1. **Hybrid Approach**: Track (passive) + Reserve (active)
2. **Platform-Native**: Swift on macOS, Rust on Linux (not Electron)
3. **Developer-First**: Built for dev workflow, not production ops
4. **Zero Configuration**: Works out of the box, no complex setup
5. **Visual Feedback**: Real-time UI showing port status

### Key Innovations

1. **Graceful Degradation**: Works without daemon (file-based locks)
2. **Intelligent Allocation**: Learns user preferences over time
3. **Multi-Developer Support**: Per-user namespaces on shared machines
4. **Conflict Resolution**: Multiple strategies (kill, redirect, notify)
5. **Historical Tracking**: Audit log for debugging

---

## Future Enhancements (Post-v1.0)

### Short Term (v1.1-v1.5)
- [ ] Port groups/pools for microservices
- [ ] Docker integration (automatic port mapping)
- [ ] Kubernetes local dev support
- [ ] Web UI (access from browser)
- [ ] Export/import port configurations

### Medium Term (v2.0)
- [ ] Distributed port management (across machines)
- [ ] Port forwarding/SSH tunnel management
- [ ] Automatic conflict resolution policies
- [ ] Integration with IDE plugins (VS Code, IntelliJ)
- [ ] Cloud sync of preferences

### Long Term
- [ ] Production-ready (beyond dev)
- [ ] Port rebalancing (load-aware allocation)
- [ ] AI-powered conflict prediction
- [ ] Integration with service meshes
- [ ] Commercial support/options

---

## Technical Risks & Mitigations

### Risk 1: SQLite Performance Bottleneck
**Probability**: Medium
**Impact**: High
**Mitigation**:
- Use WAL mode extensively
- Proper indexing
- Periodic vacuum/optimization
- Alternative: migrate to SQLite alternatives if needed (LMDB, RocksDB)

### Risk 2: Race Conditions in Port Detection
**Probability**: High
**Impact**: Medium
**Mitigation**:
- Retry mechanism with exponential backoff
- Atomic registration (check + claim in transaction)
- Accept that race conditions will occur, handle gracefully

### Risk 3: macOS App Store Restrictions
**Probability**: Low (if distributing outside App Store)
**Impact**: Low
**Mitigation**:
- Distribute signed .app directly
- Notarize for macOS 13+
- Provide clear installation instructions

### Risk 4: Linux Fragmentation (Desktop Environments)
**Probability**: High
**Impact**: Medium
**Mitigation**:
- Support libayatana-appindicator (works on most)
- Fallback to CLI-only mode
- Document per-distro requirements

### Risk 5: Low Adoption (Championing Problem)
**Probability**: Medium
**Impact**: High
**Mitigation**:
- Start with simple CLI (low friction)
- Make it useful even without registration (passive tracking)
- Integrate with popular dev tools (Create React App, Django startproject, etc.)

### Risk 6: Complexity Explosion
**Probability**: Medium
**Impact**: High
**Mitigation**:
- Strict MVP scope
- Incremental development
- Regular refactoring
- Kill features that don't prove useful

---

## Success Metrics

### v1.0 Release Goals
- [ ] <1s startup time
- [ ] <10ms registration latency
- [ ] <50MB memory footprint
- [ ] 0% data corruption in stress tests
- [ ] 95% code coverage for core logic

### Adoption Metrics
- [ ] 1000+ GitHub stars within 6 months
- [ ] 50+ active contributors
- [ ] Featured in dev communities (Hacker News, Reddit, etc.)
- [ ] Integration with 10+ popular frameworks

### User Satisfaction
- [ ] <1 minute setup time
- [ ] Zero configuration required
- [ ] Intuitive UI (no manual needed)
- [ ] Reduces "port already in use" frustration by 90%

---

## References & Resources

### Documentation & Tutorials
- SQLite: https://www.sqlite.org/wal.html
- GRDB.swift: https://github.com/philtre/GRDB.swift
- NSStatusItem: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/StatusBar/
- Unix Domain Sockets: https://tldp.org/LPG/7.13.html

### Libraries & Crates
- Rust: rusqlite, tokio, nix, clap
- Swift: GRDB, SwiftUI, Network framework
- Linux: libayatana-appindicator, tui-rs

### Tools for Inspiration
- lsof, netstat, ss (port detection)
- Docker (dynamic port allocation)
- etcd (service registry)
- Supervisor (process management)

---

## Conclusion

This document outlines a comprehensive plan for building a modern, cross-platform port management tool. The key success factors are:

1. **Simplicity**: Start with MVP, iterate based on real usage
2. **Performance**: Keep memory footprint and latency low
3. **Developer Experience**: Make it easy to adopt, integrate with existing tools
4. **Platform Integration**: Feel native on macOS and Linux
5. **Reliability**: Handle edge cases, crashes, and conflicts gracefully

The next steps are:
1. Create Beads issues for implementation phases
2. Set up CI/CD for both platforms
3. Implement Phase 1 (Core MVP)
4. Gather early feedback from beta testers
5. Iterate based on real-world usage patterns

---

*Document Version*: 1.0  
*Last Updated*: 2024-12-27  
*Status*: Planning Phase - Ready for Implementation

---

## Happy Path Analysis: Vite App Starting on Port 3000

### The Status Quo (Today)

```bash
$ pnpm dev
> VITE v5.0.0  ready in 234 ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

**If port is free**: ✅ Success  
**If port occupied**: ❌ Error: `Error: Port 3000 is already in use`

---

### Scenario Analysis

#### Scenario 1: Port Manager NOT Installed (No Change)
**User Experience**: Identical to today
- App starts normally if port free
- Fails with standard error if port occupied
- User manually kills process or changes port

**Complexity**: Zero
**Adoption**: N/A (tool doesn't exist)

---

#### Scenario 2: Port Manager Running, App NOT Integrated
**User Experience**: Identical to today
- App doesn't know about port manager
- Port manager passively observes (in background)
- No change to developer workflow

**Complexity**: Zero for user, minimal for port manager (just runs)
**Adoption**: High (tool doesn't need app to change)

**What Port Manager Does**:
```bash
# In background, port manager detects:
# "Process 12345 (node) is now listening on 3000"
# Records: Port 3000, Service "Vite", PID 12345, Started 10:30 AM
```

**User Value**: 
- Can run `freeport list` to see what's running
- Can run `freeport kill 3000` to quickly free port
- Better than `lsof -i :3000 | kill $(...)`

---

#### Scenario 3: Port Manager Running, App Integrated via Wrapper Script

**Implementation**:
```bash
# User adds to package.json:
{
  "scripts": {
    "dev": "freeport-wrapper 3000 pnpm dev"
  }
}
```

**Or uses npx/integration**:
```bash
# Simpler: User runs with npx
$ npx freeport run --port 3000 -- pnpm dev
```

**What Happens**:

**3A. Port 3000 is Free**
```
1. freeport-wrapper checks with port manager via UDS
2. Port manager: "3000 is free, go ahead"
3. Wrapper marks 3000 as "claimed for Vite"
4. Vite starts normally
5. Vite binds to 3000 successfully

User sees: Normal Vite startup
```

**Complexity**: Low (wrapper script or CLI prefix)
**Adoption**: Medium (requires changing package.json or remembering npx)

---

**3B. Port 3000 is Occupied by OLD Vite Process**
```
1. freeport-wrapper checks with port manager
2. Port manager: "3000 is in use by your Vite process (PID 12345, started 2 hours ago)"
3. Port manager asks: "Kill old Vite process and restart? [Y/n]"
4. User presses Y
5. Port manager kills PID 12345
6. Wrapper marks 3000 as "claimed for new Vite"
7. Vite starts successfully

User sees:
> Old Vite process detected on port 3000
> Kill old process? [Y/n]: Y
> Killing process 12345...
> VITE v5.0.0  ready in 234 ms
```

**Complexity**: Medium (interaction required)
**Adoption**: Medium-High (solves common problem)

---

**3C. Port 3000 is Occupied by DIFFERENT Process**
```
1. freeport-wrapper checks with port manager
2. Port manager: "3000 is in use by Python Django (PID 67890)"
3. Wrapper asks: 
   "Port 3000 is occupied by Django (PID 67890)
    Options:
    1) Use next free port (3001)
    2) Kill Django (DANGEROUS)
    3) Cancel
    Choose [1-3]: "
4. User chooses 1
5. Port manager: "3001 is free"
6. Wrapper sets environment variable VITE_PORT=3001
7. Vite starts on 3001

User sees:
> Port 3000 is occupied by Django (PID 67890)
> 
> Options:
> 1) Use next free port (3001)  <- RECOMMENDED
> 2) Kill Django (DANGEROUS)
> 3) Cancel
> 
> Choose [1-3]: 1
> VITE v5.0.0  ready in 234 ms
> 
> Using port 3001 instead of 3000
```

**Complexity**: High (multiple options)
**Adoption**: Medium (interactive complexity)

---

**3D. Simplified Version (No Interactive Choices)**
```
1. freeport-wrapper checks with port manager
2. Port manager: "3000 is occupied by Django"
3. Wrapper: "Trying next free port..."
4. Finds 3001 is free
5. Starts Vite on 3001

User sees:
> Port 3000 is occupied, using 3001 instead
> VITE v5.0.0  ready in 234 ms
> 
> Using port 3001 (configured for 3000)
```

**Complexity**: Low
**Adoption**: High (simple, just works)

---

#### Scenario 4: Port Manager Integrated into Framework (Vite Plugin)

**Implementation**:
```javascript
// vite.config.js
import freeport from '@freeport/vite'

export default {
  plugins: [
    freeport({ port: 3000 })
  ]
}
```

**What Happens**:

**4A. Port is Free**
```
1. Vite loads freeport plugin
2. Plugin checks port manager via UDS (async, <5ms)
3. Port manager: "3000 is free"
4. Vite starts normally on 3000

User sees: Normal startup (no change)
```

**Complexity**: Low (one plugin)
**Adoption**: Low-High (depends on plugin ecosystem)

---

**4B. Port is Occupied**
```
1. Plugin checks port manager
2. Port manager: "3000 is in use by old Vite (PID 12345)"
3. Plugin: Auto-kills old process (if same user)
4. Plugin starts on 3000

OR

2. Port manager: "3000 is in use by Django"
3. Plugin: Finds free port (3001, 3002, etc.)
4. Plugin updates Vite config to use new port
5. Vite starts

User sees:
> Port 3000 occupied, starting on 3001
> VITE v5.0.0  ready in 234 ms
```

**Complexity**: Low-Medium (plugin handles complexity)
**Adoption**: Medium (requires plugin per framework)

---

#### Scenario 5: Port Manager Down (Crash/Not Started)

**What Happens**:

**5A. App NOT integrated**
```
1. App tries to start
2. App can't connect to port manager (socket not found)
3. App falls back to standard behavior (bind directly)
4. Works normally, just without port manager benefits

User sees: No change (port manager was invisible anyway)
```

**Complexity**: Zero (graceful degradation)
**Adoption**: N/A

---

**5B. App IS integrated (via wrapper)**
```
1. Wrapper tries to connect to port manager
2. Socket connection fails
3. Wrapper falls back: "Port manager not available, trying direct bind..."
4. Wrapper tries to bind to 3000
5A. If free: Success
5B. If occupied: Standard error

User sees:
> Port manager not available, using direct bind
> VITE v5.0.0  ready in 234 ms

OR

> Port manager not available, using direct bind
> Error: Port 3000 is already in use
```

**Complexity**: Low (fallback logic)
**Adoption**: High (tool is optional)

---

#### Scenario 6: Race Condition (Two Apps Start Simultaneously)

**What Happens**:

**6A. Without Port Manager**
```
1. App A tries to bind 3000 -> Success
2. App B tries to bind 3000 -> Error "already in use"
3. Developer fixes manually
```

**6B. With Port Manager (Reservation Mode)**
```
1. App A checks port manager -> "3000 free"
2. Port manager marks 3000 as "reserved for App A"
3. App B checks port manager -> "3000 reserved by App A"
4. App B finds 3001 is free -> "reserved for App B"
5. Both apps start successfully

User sees:
> App A: Vite v5.0.0  ready in 234 ms (port 3000)
> App B: Vite v5.0.0  ready in 234 ms (port 3001)
```

**Complexity**: Medium (race condition handling)
**Adoption**: High (prevents conflicts)

**6C. With Port Manager (Try-Bind Fallback)**
```
1. App A tries to bind 3000 -> Success
2. App B tries to bind 3000 -> Error "EADDRINUSE"
3. Port manager detects: "3000 occupied by App A"
4. Port manager finds 3001 is free
5. Port manager tells App B: "Use 3001"
6. App B starts on 3001

User sees:
> App B: Port 3000 occupied, using 3001 instead
> VITE v5.0.0  ready in 234 ms
```

**Complexity**: Medium
**Adoption**: High

---

#### Scenario 7: App Crashes (Zombie Port)

**What Happens**:

**7A. Without Port Manager**
```
1. Vite starts on 3000
2. Vite crashes (unhandled exception)
3. Port 3000 stays in TIME_WAIT
4. User runs `pnpm dev` again -> Error "already in use"
5. User runs `lsof -i :3000` -> sees nothing or TIME_WAIT
6. User waits 60 seconds or runs `kill -9` manually
```

**Complexity**: High for user (confusing, requires commands)
**Adoption**: N/A

---

**7B. With Port Manager (Passive)**
```
1. Vite starts on 3000
2. Port manager records: "Vite on 3000, PID 12345"
3. Vite crashes (PID 12345 dies)
4. Port manager's cleanup daemon (every 30s):
   - Checks PID 12345 -> not running
   - Marks port 3000 as "stale"
   - Removes entry from database
5. User runs `pnpm dev` again
6. Port manager: "3000 is free"
7. Vite starts successfully

User sees: No issues, app restarts normally
```

**Complexity**: Low (automatic cleanup)
**Adoption**: High (invisible benefit)

---

**7C. With Port Manager (Active Registration)**
```
1. Vite plugin registers with port manager
2. Port manager: "3000 reserved for Vite"
3. Vite starts on 3000
4. Vite crashes (sends no cleanup signal)
5. Heartbeat missed (60s timeout)
6. Port manager: "Vite heartbeat missed, releasing port 3000"
7. User runs `pnpm dev` again
8. Port manager: "3000 is free"
9. Vite starts successfully

User sees: No issues, app restarts normally
```

**Complexity**: Low (automatic cleanup)
**Adoption**: High (invisible benefit)

---

#### Scenario 8: Multi-Developer on Shared Machine

**What Happens**:

**8A. Without Port Manager**
```
Developer A: Starts Vite on 3000
Developer B: Starts Vite on 3000 -> Error "already in use"
Developer B: Changes port to 3001
Developer A: Forgets about their Vite, leaves running
Developer B: Can't use 3000 because A's process is still running
Developer B: Has to find A and ask them to kill their process
```

**Complexity**: High (requires coordination)
**Adoption**: N/A

---

**8B. With Port Manager (Per-User Namespaces)**
```
Developer A (user alice): 
- freeport list (alice's namespace):
  - Port 3000: Vite (alice) [active]
  
Developer B (user bob):
- freeport list (bob's namespace):
  - (nothing)

Developer B runs `pnpm dev`:
1. Wrapper checks port manager
2. Port manager: "3000 is in use by alice@Vite (not in your namespace)"
3. Wrapper: "Your ports are free, starting on 3001"
4. Vite starts on 3001

Developer A later:
- freeport list:
  - Port 3000: Vite (alice) [stale - crashed 2h ago]
- freeport cleanup
- Port 3000 freed
```

**Complexity**: Medium
**Adoption**: High for shared environments (rare in practice)

---

#### Scenario 9: Docker Integration

**What Happens**:

**9A. Without Port Manager**
```
docker run -p 3000:3000 my-app
docker run -p 3000:3000 my-app  -> Error "bind: address already in use"
```

---

**9B. With Port Manager**
```
# Developer uses wrapper
freeport-docker run -p 3000:3000 my-app

1. Wrapper checks port manager
2. Port manager: "3000 is occupied by local Vite"
3. Wrapper: "Using host port 3001 instead"
4. docker run -p 3001:3000 my-app

OR

Wrapper: "Port 3000 free, proceeding"
docker run -p 3000:3000 my-app
```

**Complexity**: Medium (needs Docker integration)
**Adoption**: Low (Docker handles ports differently)

---

#### Scenario 10: CI/CD Pipeline

**What Happens**:

**10A. Without Port Manager**
```
# CI job 1: Run tests on port 3000
# CI job 2: Run tests on port 3000
# Jobs might conflict if running on same runner
```

---

**10B. With Port Manager**
```
# CI configuration
- name: Setup freeport
  run: |
    freeport reserve --pool test-ports --job-id $CI_JOB_ID

- name: Run tests
  run: pnpm test
  env:
    PORT: 3000  # freeport manages allocation

1. Job 1 reserves first available port (e.g., 3000)
2. Job 2 reserves next available port (e.g., 3001)
3. Both run without conflict
```

**Complexity**: Medium (CI integration)
**Adoption**: High for CI (common pain point)

---

## Simplicity Assessment

### Complexity Scoring

| Scenario | User Complexity | Integration Complexity | Value | Recommendation |
|----------|----------------|----------------------|--------|----------------|
| 1. No port manager | 0 | 0 | 0 | Baseline |
| 2. Passive observation | 0 | Low | Medium | ✅ Phase 1 |
| 3A. Wrapper (free port) | Low | Low | Low | ⚠️ Maybe |
| 3B. Wrapper (old process) | Medium | Low | High | ✅ Phase 2 |
| 3C. Wrapper (different process) | High | Low | Medium | ❌ Too complex |
| 3D. Wrapper (auto-fallback) | Low | Low | High | ✅ Phase 1 |
| 4. Framework plugin | Low | Medium | High | ✅ Phase 3 |
| 5. Port manager down | 0 | Low | N/A | ✅ Always support |
| 6. Race conditions | Low | Medium | High | ✅ Phase 2 |
| 7. Zombie cleanup | 0 | Low | High | ✅ Phase 1 |
| 8. Multi-dev | Medium | Medium | Low | ⚠️ Rare use case |
| 9. Docker | Low | Medium | Medium | ⚠️ Maybe later |
| 10. CI/CD | Low | Medium | High | ✅ Phase 2 |

---

### Simplicity Recommendations

#### ✅ **Keep SIMPLE (Phase 1)**

**Start with Passive Mode Only:**
```bash
# Port manager just runs in background
# No integration required
# Just provides better visibility and tools

$ freeport list
Port 3000: Vite (PID 12345) [active] - Started 10:30 AM
Port 5000: Django (PID 67890) [active] - Started 9:00 AM
Port 8080: Java Server (PID 54321) [stale] - Crashed 2h ago

$ freeport kill 3000
Killing Vite process (PID 12345)...
Port 3000 freed
```

**Value**:
- Better than `lsof | kill`
- Automatic zombie cleanup
- Simple CLI for common tasks
- Zero integration needed

---

#### ✅ **Add Wrapper Script (Phase 2)**

**Simple wrapper that auto-falls back:**
```bash
# package.json
"scripts": {
  "dev": "freeport run --port 3000 --fallback -- pnpm dev"
}

# Or even simpler
$ npx @freeport/run --port 3000 -- pnpm dev
```

**Behavior**:
- Try to use preferred port
- If occupied, find next free port automatically
- No interactive prompts (keep it fast)
- Graceful fallback if port manager down

**Value**:
- Solves "port already in use" with one command
- Works without port manager
- Minimal adoption friction

---

#### ❌ **AVOID for v1.0**

**Interactive Prompts** (Scenario 3C):
- Too much friction
- Breaks automation
- Can use flags instead (--auto-kill, --find-next, etc.)

**Per-User Namespaces** (Scenario 8):
- Rarely needed in practice
- Adds complexity
- Can add later if requested

**Framework Plugins** (Scenario 4):
- Different plugin per framework
- High maintenance burden
- Start with wrapper scripts (universal)

**Complex Conflict Resolution**:
- Keep it simple: "next free port" or "fail"
- Don't ask users to make complex decisions

---

### Revised Architecture for Simplicity

#### **Phase 1: Passive Observation + CLI (MVP)**

```
Components:
1. Daemon (runs in background)
   - Scans ports every 30s
   - Stores in SQLite
   - Cleans up stale processes
   
2. CLI tool
   - list: Show all ports
   - kill: Kill process on port
   - cleanup: Remove stale entries
   
3. Zero integration required
```

**Happy Path (Phase 1):**
```bash
# User just installs tool and forgets it
$ brew install freeport  # or cargo install freeport

# Tool auto-starts on login (macOS LaunchAgent, Linux systemd)

# Developer workflow UNCHANGED:
$ pnpm dev
# If port free: success
# If port occupied: standard error

# But now developer has better tools:
$ freeport list  # See what's running
$ freeport kill 3000  # Quick kill (better than lsof | kill)
$ freeport cleanup  # Remove zombie ports
```

**Value**:
- Better visibility
- Easier conflict resolution
- Automatic zombie cleanup
- Zero adoption friction

---

#### **Phase 2: Simple Wrapper + Smart Port Selection**

```
Components (add to Phase 1):
4. Wrapper script / npx tool
   - Check preferred port
   - If free: use it
   - If occupied: find next free port
   - Fall back to direct bind if port manager down
```

**Happy Path (Phase 2):**
```bash
# User optionally adds wrapper to package.json
{
  "scripts": {
    "dev": "freeport run --port 3000 -- pnpm dev"
  }
}

# Or uses npx (no package.json change)
$ npx @freeport/run --port 3000 -- pnpm dev

# What happens:
# - If 3000 free: Starts on 3000
# - If 3000 occupied: Starts on 3001 (or next free)
# - Shows message: "Using port 3001 instead of 3000"
```

**Value**:
- Solves port conflicts automatically
- Minimal adoption cost
- Works without port manager (graceful degradation)

---

#### **Phase 3: Framework Plugins (Optional)**

```
Only if Phase 2 is popular:
5. Vite plugin, Create React App plugin, etc.
   - Same logic as wrapper
   - More seamless integration
```

**Value**:
- Even simpler for users (no script changes)
- Requires maintenance per framework

---

## Final Recommendation

### **Start with Phase 1 (Passive + CLI)**
- Zero adoption friction
- Immediate value (better than existing tools)
- Low implementation complexity
- Can ship in 2-3 weeks

### **Add Phase 2 (Simple Wrapper)**
- If Phase 1 gets users asking for automation
- High value, low complexity
- Can ship in 1-2 weeks after Phase 1

### **Phase 3+ Only If Needed**
- Framework plugins (Phase 3)
- Per-user namespaces (Phase 4)
- CI/CD integration (Phase 5)
- Docker integration (Phase 6)

---

## Simplicity Summary

**The key insight**: Most complexity comes from ACTIVE mode (apps registering). 

**PASSIVE mode provides 80% of value with 20% of complexity:**
- Better visibility than `lsof`
- Easier kill than `lsof | kill`
- Automatic zombie cleanup
- Works immediately without integration

**Simple wrapper provides 90% of value with 30% of complexity:**
- Automatic port selection
- Solves most conflict pain points
- One-line adoption (npx)
- Graceful fallback

**Interactive prompts and complex conflict resolution add complexity without proportional value:**
- Breaks automation
- Slows down workflow
- Users prefer fast, automatic solutions

**Conclusion**: Start passive, add simple wrapper later, keep it simple!

