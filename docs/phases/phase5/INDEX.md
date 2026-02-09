# Phase 5: Live Watcher - Complete Implementation Guide

## Welcome to Phase 5! 🚀

This folder contains **detailed implementation guides** for each task in Phase 5. Each file is a complete deep-dive with:
- Problem overview
- Architecture decisions
- Complete code implementations
- Testing strategies
- Critical points and common mistakes
- Checklist for completion

---

## Quick Navigation

### 📋 Tasks (in order)

1. **[TASK_5_1_FILE_WATCHER.md](./TASK_5_1_FILE_WATCHER.md)** — Implement File Watcher
   - fsnotify integration
   - Recursive directory watching
   - Exclusion pattern support
   - ~2-3 hours effort

2. **[TASK_5_2_EVENT_DEBOUNCING.md](./TASK_5_2_EVENT_DEBOUNCING.md)** — Implement Event Debouncing
   - 500ms debounce window
   - Event batching
   - Hash comparison to skip redundant rebuilds
   - ~2 hours effort

3. **[TASK_5_3_WATCH_ORCHESTRATION.md](./TASK_5_3_WATCH_ORCHESTRATION.md)** — Implement Watch Orchestration
   - Rebuild trigger logic
   - Single rebuild at a time
   - Status messages
   - ~2-3 hours effort

4. **[TASK_5_4_WATCH_COMMAND.md](./TASK_5_4_WATCH_COMMAND.md)** — Implement Watch CLI Command
   - `kudev watch` command
   - Integration with up command
   - User feedback
   - ~2 hours effort

**Total Effort**: ~8-10 hours  
**Total Complexity**: 🟡 Intermediate (file watching, event handling)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│              File System Events                      │
│         (create, modify, delete files)               │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                   File Watcher                       │
│              (fsnotify wrapper)                      │
│                    Task 5.1                          │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                   Debouncer                          │
│              (500ms window)                          │
│                    Task 5.2                          │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                  Orchestrator                        │
│              (rebuild trigger)                       │
│                    Task 5.3                          │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                Build → Deploy                        │
│              (Phase 2 + 3)                          │
└─────────────────────────────────────────────────────┘
```

### Watch Flow

```
User saves file
    │
    ▼
fsnotify detects change
    │
    ▼
File excluded? ──Yes──► Ignore
    │
    No
    ▼
Add to debounce batch
    │
    ▼
Wait 500ms for more events
    │
    ▼
Calculate new hash
    │
    ▼
Hash same as before? ──Yes──► Skip rebuild
    │
    No
    ▼
Rebuild and redeploy
    │
    ▼
"Ready" message
```

---

## Dependency Flow

```
Phase 1-4 (Config, Build, Deploy, CLI)
    ↓
Task 5.1 (File Watcher)
    ↓
Task 5.2 (Debouncer)
    ↓
Task 5.3 (Orchestrator)
    ↓
Task 5.4 (Watch Command)
    ↓
Phase 6 (Testing)
```

---

## Key Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Watch library | fsnotify | Standard Go file watching |
| Debounce window | 500ms | Balance between responsiveness and batching |
| Rebuild check | Hash comparison | Skip unnecessary rebuilds |
| Concurrency | One rebuild at a time | Prevent queue buildup |

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `pkg/watch/watcher.go` | File watching | `Watcher`, `Watch()` |
| `pkg/watch/debounce.go` | Event batching | `Debouncer`, `Debounce()` |
| `pkg/watch/orchestrator.go` | Rebuild logic | `Orchestrator`, `Run()` |
| `cmd/commands/watch.go` | Watch command | `kudev watch` |

---

## Testing Strategy

### Unit Tests

| File | Coverage Target | Focus |
|------|-----------------|-------|
| `pkg/watch/watcher_test.go` | 75%+ | Event detection, exclusions |
| `pkg/watch/debounce_test.go` | 85%+ | Timing, batching |
| `pkg/watch/orchestrator_test.go` | 70%+ | Rebuild triggering |

---

## Quick Start Checklist

Before starting Phase 5, ensure Phase 1-4 are complete:
- [ ] Config loading working
- [ ] Docker builder working
- [ ] K8s deployer working
- [ ] `kudev up` command working
- [ ] Graceful shutdown working

---

## Common Mistakes to Avoid

1. **Watching too many files** — Use directory watching, not individual files
2. **No debouncing** — Multiple saves = multiple rebuilds
3. **Rebuilding on excluded files** — Check exclusions first
4. **Queueing rebuilds** — Only one rebuild at a time
5. **Ignoring hash check** — Rebuild only when source changed

---

## References

- [fsnotify Documentation](https://pkg.go.dev/github.com/fsnotify/fsnotify)
- [Go Timer](https://pkg.go.dev/time#Timer)
- [Debouncing Pattern](https://en.wikipedia.org/wiki/Debouncing)

---

**Next**: Start with [TASK_5_1_FILE_WATCHER.md](./TASK_5_1_FILE_WATCHER.md) 🚀

