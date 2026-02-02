# Phase 5: Live Watcher (Hot Reload)

**Objective**: Watch for local file changes and automatically rebuild/redeploy, enabling fast feedback loop during development.

**Timeline**: 1 week  
**Difficulty**: 🟡 Intermediate (file watching, event debouncing)  
**Dependencies**: Phase 1-4 (all previous phases)

---

## 📋 Quick Overview

Features in this phase:

1. **File Watcher** — Monitor source files using `fsnotify`
2. **Event Debouncing** — Batch file events within 500ms window
3. **Rebuild Trigger** — On file change, re-run full `kudev up` pipeline
4. **Watch Command** — CLI command to start hot-reload mode
5. **User Feedback** — Clear "Watching...", "Rebuilding...", "Ready" messages

---

## 📝 Core Tasks

### Task 5.1: Implement File Watcher

**Files**:
- `pkg/watch/watcher.go` — Watcher interface and implementation

**Key Points**:
- Watch source directory recursively
- Ignore patterns: `.git`, `node_modules`, `.kudev.yaml`, `vendor`
- Report file changes back to caller
- Respect context cancellation
- Clear error messages if watcher init fails

**Interface**:
```go
type Watcher interface {
    Watch(ctx context.Context, sourceDir string) (<-chan FileChangeEvent, error)
    Close() error
}

type FileChangeEvent struct {
    Path     string
    Op       string  // "write", "create", "delete"
    Timestamp time.Time
}
```

**Implementation Hints**:
- Use `github.com/fsnotify/fsnotify` library
- Watch on parent dirs only (avoid watching thousands of files)
- Load exclusion patterns from `.dockerignore` or `.kudevignore`
- Handle "too many open files" error on large projects

---

### Task 5.2: Implement Event Debouncing

**Files**:
- `pkg/watch/debounce.go` — Debouncing logic

**Key Points**:
- Collect events for 500ms
- Only trigger rebuild when batch is complete
- Prevents rapid rebuilds on multi-file saves
- Skip rebuild if source hash unchanged

**Debounce Algorithm**:
```
1. Start 500ms timer when first event received
2. Collect all events during timer window
3. When timer fires:
   a. Calculate new source hash
   b. If hash unchanged: skip rebuild
   c. If hash changed: trigger rebuild
4. Reset and wait for next event batch
```

**Implementation Hints**:
- Use `time.Timer` with `<-timer.C` channel
- Only trigger rebuild if source hash actually changed
- Log debounce decisions at debug level
- Make debounce window configurable (default 500ms)

---

### Task 5.3: Implement Watch Orchestration

**Files**:
- `pkg/watch/orchestrator.go` — Rebuild trigger and sequencing

**Key Points**:
- Watch for file changes
- Debounce events
- Check if rebuild needed (hash comparison)
- Run full `kudev up` pipeline on change
- Only one rebuild in progress at a time
- Clear status messages

**Orchestrator Pattern**:
```go
type Orchestrator struct {
    watcher    Watcher
    debouncer  *Debouncer
    builder    Builder
    deployer   Deployer
    logger     Logger
}

func (o *Orchestrator) Watch(ctx context.Context) error {
    // 1. Start file watcher
    // 2. Listen for file changes
    // 3. Debounce events
    // 4. Trigger rebuilds
    // 5. Print status
}
```

**Success Criteria**:
- ✅ File changes detected and reported
- ✅ Debouncing works (multiple events = single rebuild)
- ✅ Hash comparison prevents unnecessary rebuilds
- ✅ One rebuild at a time (no queue stacking)
- ✅ Clear user feedback

---

### Task 5.4: Implement Watch CLI Command

**Files**:
- `cmd/watch.go` — Watch command

**Usage**:
```bash
kudev watch
# Output:
# Watching for changes in /path/to/project
# Press Ctrl+C to stop
# [Watching for changes...]
# [File modified: main.go]
# [Rebuilding...]
# [Building Docker image...]
# [Deploying to cluster...]
# [Port forwarding on :8080]
# [Ready]
```

**Success Criteria**:
- ✅ Starts file watcher
- ✅ Shows "Watching for changes..." message
- ✅ Rebuilds and deploys on file changes
- ✅ Streams logs after each deployment
- ✅ Clear "Ready" message when stable
- ✅ Ctrl+C stops cleanly

---

## 🧪 Testing Strategy

- Mock file watcher events
- Test debouncing with simulated rapid events
- Test hash comparison skip logic
- Test graceful context cancellation
- Verify one-rebuild-at-a-time enforcement

**Test Coverage**: 70%+

---

## ✅ Success Criteria

- ✅ File changes trigger rebuilds automatically
- ✅ Multiple changes debounced into single rebuild
- ✅ Hash comparison prevents unnecessary rebuilds
- ✅ Clear "Watching for changes..." feedback
- ✅ `kudev watch` command works end-to-end
- ✅ Ctrl+C stops watching gracefully

---

**Next**: [Phase 6 - Testing & Reliability](./PHASE_6_TESTING_RELIABILITY.md) ✅
