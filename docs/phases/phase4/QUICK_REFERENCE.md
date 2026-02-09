# Phase 4 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 4. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 4.1 (4h)  → Log tailing
Task 4.2 (4h)  → Port forwarding
Task 4.3 (4h)  → CLI orchestration (up/down/status)
Task 4.4 (2h)  → Graceful shutdown
         ────────
Total: ~12-14 hours
```

---

## Core Concepts

### 1. Complete Developer Loop
- **`kudev up`** — Build → Deploy → Port Forward → Stream Logs
- **`kudev down`** — Clean deletion
- **`kudev status`** — Show deployment health

### 2. Background Services
- Port forwarding runs in goroutine
- Log streaming blocks main thread
- Ctrl+C stops everything cleanly

### 3. Signal Handling
- SIGINT/SIGTERM → Cancel context
- Double Ctrl+C → Force exit
- Cleanup via defer

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `pkg/logs/tailer.go` | Log streaming | `LogTailer`, `TailLogs()` |
| `pkg/logs/discovery.go` | Pod discovery | `PodDiscovery`, `DiscoverPod()` |
| `pkg/portfwd/forwarder.go` | Port forward | `PortForwarder`, `Forward()` |
| `cmd/commands/up.go` | Up command | Build→Deploy→Logs |
| `cmd/commands/down.go` | Down command | Delete resources |
| `cmd/commands/status.go` | Status command | Show health |

---

## Key Patterns

### Log Tailing

```go
tailer := logs.NewKubernetesLogTailer(clientset, logger, os.Stdout)
err := tailer.TailLogsWithRetry(ctx, appName, namespace)
// Blocks until ctx cancelled
```

### Port Forwarding

```go
forwarder := portfwd.NewKubernetesPortForwarder(clientset, restConfig, logger)
err := forwarder.Forward(ctx, appName, namespace, localPort, podPort)
// Returns immediately, runs in background
defer forwarder.Stop()
```

### Signal Handling

```go
ctx, cancel := context.WithCancel(context.Background())
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigChan
    cancel()
}()
```

### Cleanup Pattern

```go
var cleanups []func()
defer func() {
    for _, fn := range cleanups {
        fn()
    }
}()

cleanups = append(cleanups, forwarder.Stop)
```

---

## Implementation Checklist

### Task 4.1: Log Tailing
```
[ ] pkg/logs/discovery.go
[ ] pkg/logs/tailer.go
[ ] PodDiscovery.DiscoverPod()
[ ] LogTailer.TailLogs()
[ ] TailLogsWithRetry() for reconnection
```

### Task 4.2: Port Forwarding
```
[ ] pkg/portfwd/forwarder.go
[ ] checkPortAvailable()
[ ] SuggestAlternativePort()
[ ] Forward() with SPDY
[ ] Stop() for cleanup
```

### Task 4.3: CLI Orchestration
```
[ ] cmd/commands/up.go
[ ] cmd/commands/down.go
[ ] cmd/commands/status.go
[ ] --no-logs, --no-port-forward flags
[ ] --force flag for down
[ ] --watch flag for status
```

### Task 4.4: Graceful Shutdown
```
[ ] Signal handling in root.go
[ ] Context cancellation
[ ] Double Ctrl+C force exit
[ ] Cleanup defer pattern
```

---

## Command Reference

### kudev up
```bash
kudev up                    # Full build and deploy
kudev up --no-logs          # Skip log streaming
kudev up --no-port-forward  # Skip port forward
kudev up --no-build         # Skip build (use existing image)
```

### kudev down
```bash
kudev down         # With confirmation
kudev down --force # Skip confirmation
```

### kudev status
```bash
kudev status          # One-time status
kudev status --watch  # Continuous monitoring
```

---

## Common Commands

```bash
# Run all Phase 4 tests
go test ./pkg/logs/... ./pkg/portfwd/... ./cmd/... -v

# Build CLI
go build -o kudev ./cmd/main.go

# Test up command
./kudev up

# Test graceful shutdown
./kudev up
# Press Ctrl+C
```

---

## User Experience Flow

```
$ kudev up
✓ Loading configuration...
✓ Calculating source hash...
✓ Building image myapp:kudev-a1b2c3d4...
✓ Loading image to cluster...
✓ Deploying to Kubernetes...
✓ Waiting for pods to be ready...
✓ Port forwarding localhost:8080 → pod:8080

═══════════════════════════════════════════════════
  Application is running!
  Local:   http://localhost:8080
  Status:  Running (2/2 replicas)
═══════════════════════════════════════════════════

Streaming logs (Ctrl+C to stop)...

[2024-01-15 10:30:45] Starting server on :8080
[2024-01-15 10:30:46] Ready to accept connections
^C
Cleaning up...
✓ Port forward stopped
✓ Deployment remains running (use 'kudev down' to remove)
```

---

## Next Phase

After completing Phase 4:
- ✅ Can build, deploy, and access application
- ✅ Can stream logs in real-time
- ✅ Can forward ports locally
- ✅ Graceful shutdown works

**Phase 5 (Live Watcher)** will:
- Watch for file changes
- Auto-rebuild on change
- Debounce rapid changes

