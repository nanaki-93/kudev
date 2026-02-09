# Phase 5 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 5. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 5.1 (3h)  → File watcher (fsnotify)
Task 5.2 (2h)  → Event debouncing
Task 5.3 (3h)  → Watch orchestration
Task 5.4 (2h)  → Watch CLI command
         ────────
Total: ~8-10 hours
```

---

## Core Concepts

### 1. File Watching
- **Library**: fsnotify
- **Strategy**: Watch directories, not files
- **Exclusions**: .git, node_modules, etc.

### 2. Event Debouncing
- **Window**: 500ms
- **Purpose**: Batch rapid saves
- **Result**: Single rebuild per save-all

### 3. Hash Comparison
- **Check**: Before each rebuild
- **Skip**: If hash unchanged
- **Benefit**: No wasted builds

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `pkg/watch/watcher.go` | File watching | `FSWatcher`, `Watch()` |
| `pkg/watch/debounce.go` | Event batching | `Debouncer`, `Debounce()` |
| `pkg/watch/orchestrator.go` | Rebuild logic | `Orchestrator`, `Run()` |
| `cmd/commands/watch.go` | CLI command | `kudev watch` |

---

## Key Patterns

### File Watcher

```go
watcher, _ := NewFSWatcher(exclusions, logger)
events, _ := watcher.Watch(ctx, sourceDir)

for event := range events {
    fmt.Printf("Changed: %s (%s)\n", event.Path, event.Op)
}
```

### Debouncer

```go
debouncer := NewDebouncer(DefaultDebounceConfig(), logger)
batches := debouncer.Debounce(ctx, events)

for batch := range batches {
    fmt.Printf("Batch of %d events\n", len(batch))
    // Trigger rebuild
}
```

### Orchestrator

```go
orchestrator, _ := watch.NewOrchestrator(watch.OrchestratorConfig{
    Config:   cfg,
    Builder:  builder,
    Deployer: deployer,
    Registry: registry,
    Logger:   logger,
})

orchestrator.Run(ctx)  // Blocks until cancelled
```

---

## Implementation Checklist

### Task 5.1: File Watcher
```
[ ] pkg/watch/watcher.go
[ ] FSWatcher with fsnotify
[ ] Directory-level watching
[ ] Exclusion patterns
[ ] New directory handling
```

### Task 5.2: Debouncing
```
[ ] pkg/watch/debounce.go
[ ] 500ms debounce window
[ ] Timer reset on new events
[ ] Event batching
[ ] Non-blocking trigger
```

### Task 5.3: Orchestration
```
[ ] pkg/watch/orchestrator.go
[ ] Hash comparison
[ ] Single rebuild at a time
[ ] Queued rebuild handling
[ ] Status messages
```

### Task 5.4: Watch Command
```
[ ] cmd/commands/watch.go
[ ] Initial build/deploy
[ ] Port forwarding
[ ] Background log streaming
[ ] Orchestrator integration
```

---

## Command Reference

### kudev watch
```bash
kudev watch                    # Full watch mode
kudev watch --no-logs          # No log streaming
kudev watch --no-port-forward  # No port forward
```

---

## Watch Flow

```
File saved
    │
    ▼
fsnotify event
    │
    ▼
Excluded? ──Yes──► Ignore
    │
    No
    ▼
Add to debounce batch
    │
    ▼
[Wait 500ms]
    │
    ▼
Calculate new hash
    │
    ▼
Hash same? ──Yes──► Skip
    │
    No
    ▼
Build → Load → Deploy
    │
    ▼
"Ready" message
```

---

## Dependencies

```bash
# Add fsnotify
go get github.com/fsnotify/fsnotify
```

---

## Common Commands

```bash
# Run all Phase 5 tests
go test ./pkg/watch/... -v

# Build and test watch
go build -o kudev ./cmd/main.go
./kudev watch
```

---

## User Experience

```
$ kudev watch
✓ Loading configuration...
✓ Doing initial build and deploy...
✓ Deployed: Running (2/2 replicas)
✓ Port forwarding localhost:8080 → pod:8080

═══════════════════════════════════════════════════
  Application is running!
  Local:   http://localhost:8080
═══════════════════════════════════════════════════

Watching for changes...

# Edit file...

═══════════════════════════════════════════════════
  Change detected! Rebuilding...
═══════════════════════════════════════════════════

Building...
✓ Rebuild complete in 5.2s

Watching for changes...
```

---

## Next Phase

After completing Phase 5:
- ✅ File watching works
- ✅ Debouncing prevents rapid rebuilds
- ✅ Hash check skips unnecessary rebuilds
- ✅ `kudev watch` command works

**Phase 6 (Testing & Reliability)** will:
- Add custom error types
- Improve test coverage
- Add integration tests
- Set up CI/CD

