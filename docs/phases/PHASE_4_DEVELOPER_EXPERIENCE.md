# Phase 4: Developer Experience (Feedback & UX)

**Objective**: Close the feedback loop by automatically streaming logs, forwarding ports, and providing real-time status.

**Timeline**: 1 week  
**Difficulty**: 🟡 Intermediate (goroutines, port forwarding, streaming)  
**Dependencies**: Phase 1-3 (all previous phases)

---

## 📋 Quick Overview

Key features in this phase:

1. **Log Tailing** — Automatically stream pod logs to terminal after deployment
2. **Port Forwarding** — Background goroutine forwards local port to pod
3. **Orchestration** — `kudev up` coordinates build → deploy → logs → portfwd
4. **Status Command** — Show deployment health and pod status
5. **Graceful Shutdown** — Ctrl+C stops everything cleanly

---

## 📝 Core Tasks

### Task 4.1: Implement Log Tailing

**Files**:
- `pkg/logs/tailer.go` — LogTailer interface and implementation
- `pkg/logs/discovery.go` — Pod discovery by label selector

**Key Points**:
- Discover pods by label selector: `app: {appname}`
- Wait for pods to exist (with timeout)
- Stream logs from first pod
- Follow logs in real-time (`--follow` equivalent)
- Graceful shutdown on context cancellation

**Interface**:
```go
type LogTailer interface {
    TailLogs(ctx context.Context, appName, namespace string) error
}
```

**Implementation Hints**:
- Use `client-go/kubernetes/corev1` logs API
- Use `io.Copy` to stream (don't buffer entire output)
- Wait for pods with exponential backoff (max 5min)
- Handle pod restarts with `--tail=100` to get recent logs

---

### Task 4.2: Implement Port Forwarding

**Files**:
- `pkg/portfwd/forwarder.go` — Port forwarder implementation

**Key Points**:
- Wait for pods to be ready
- Open local port listener (default :8080)
- Forward traffic to pod container port
- Run in background goroutine
- Handle port-already-in-use errors
- Graceful shutdown

**Interface**:
```go
type PortForwarder interface {
    Forward(ctx context.Context, appName, namespace string, localPort, podPort int32) error
}
```

**Implementation Hints**:
- Use `client-go/tools/portforward` (same as kubectl)
- Check port availability before starting
- Suggest alternative port if in use
- Log "Port forward ready" when listening

---

### Task 4.3: Integrate into CLI Commands

**Files**:
- `cmd/up.go` — Orchestrate build + deploy + logs + portfwd
- `cmd/down.go` — Clean deletion
- `cmd/status.go` — Deployment status

**Up Command Flow**:
```
1. Load config
2. Validate context
3. Build image (Phase 2)
4. Load image to cluster (Phase 2)
5. Deploy to K8s (Phase 3)
6. Start log tailing (background or foreground)
7. Start port forwarding (background)
8. Wait for Ctrl+C
9. Cleanup on shutdown
```

**Success Criteria**:
- ✅ Logs stream immediately after deployment
- ✅ Port forwarding works transparently
- ✅ User can Ctrl+C to stop
- ✅ Clear status messages at each step
- ✅ Helpful errors with suggestions

---

## 🧪 Testing Strategy

- Mock log tailing with buffered output
- Mock port forwarder with test listener
- Test graceful shutdown with context cancellation
- Test error handling for port conflicts

**Test Coverage**: 75%+

---

## ✅ Success Criteria

- ✅ Logs stream to terminal after deploy
- ✅ Port forwarding works to pod port
- ✅ `kudev up` orchestrates all steps
- ✅ `kudev down` cleanly deletes
- ✅ `kudev status` shows accurate info
- ✅ Ctrl+C stops everything gracefully
- ✅ Clear status messages

---

**Next**: [Phase 5 - Live Watcher](./PHASE_5_LIVE_WATCHER.md) 👀
