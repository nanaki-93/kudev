# Task 1.2: Implement Configuration Validation

## Overview

This task implements **validation rules** that ensure `.kudev.yaml` configurations are correct before execution. Bad configurations should fail fast with **clear, actionable error messages**.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (regex, error handling, edge cases)  
**Dependencies**: Task 1.1 (Config Types)  
**Files to Create**: 
- `pkg/config/validation.go` — Validation logic
- `pkg/config/errors.go` — Custom error types
- `pkg/config/validation_test.go` — Tests

---

## The Problem Validation Solves

Without validation, users get cryptic errors much later:
```bash
$ kudev up
# 30 seconds later...
ERROR: building Docker image: error parsing Dockerfile...
# Which line in the Dockerfile? Hard to trace back to config

# With validation:
$ kudev up
ERROR: spec.dockerfilePath "./Dockerfile" does not exist
# Clear, immediate feedback!
```

---

## Validation Strategy

### Multi-Layer Validation

**Layer 1: Type Validation** (Go compiler, already done)
- YAML parser ensures correct types
- Ports are int32 (not strings)
- Replicas is positive

**Layer 2: Required Fields** (this task)
- Check mandatory fields exist
- e.g., `metadata.name` cannot be empty

**Layer 3: Format Validation** (this task)
- DNS-1123 names (lowercase, hyphens)
- Valid port ranges (1-65535)
- Valid paths (file exists)

**Layer 4: Context Validation** (Task 1.4)
- K8s context exists and is whitelisted
- Kubeconfig is valid

**Layer 5: File System Validation** (this task, extended)
- Dockerfile exists
- Project root discoverable

### Fail-Fast Philosophy

- Validate **immediately** after load
- Return **all errors** at once (not just first)
- Include **examples** in error messages
- Suggest **fixes** when possible

---

## Key Validation Decisions

### Decision 1: Fail-on-all-errors vs. Fail-fast

**Question**: Should we report first error or all errors?

**Answer**: **Report all errors at once**
- User doesn't have to iterate fixing one at a time
- Addresses root causes faster
- Better user experience

### Decision 2: When to validate file existence

**Question**: Should `dockerfilePath` validation check file exists?

**Answer**: **Contextual**
- In `Validate()`: Format check only (path looks reasonable)
- In `ValidateWithContext()`: File existence check (need project root)
- In loader: Full validation after discovering project root

### Decision 3: Strict vs. permissive port validation

**Question**: Allow privileged ports (<1024)?

**Answer**: **Allow but don't warn** (Phase 4 can add warnings)
- User might want port 80/443
- Docker Desktop handles privilege
- Better to be permissive in Phase 1

---

## Critical Points

### 1. Regex Patterns Must Be Tested

❌ Wrong pattern allows "a--b":
```go
`^[a-z0-9-]+$`  // Bad: allows consecutive hyphens

✅ Right pattern only allows single hyphens:
`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`  // Correct
```

### 2. Error Messages Must Have Examples

❌ Unhelpful error:
```
metadata.name: invalid format
```

✅ Helpful error:
```
metadata.name: must be DNS-1123 compliant
Example:
  metadata:
    name: my-app
```

### 3. Duplicate Environment Variables

Must track seen names to detect duplicates:
```go
seenNames := make(map[string]bool)
for _, v := range spec.Env {
    if seenNames[v.Name] {
        return error("duplicate")
    }
    seenNames[v.Name] = true
}
```

---

## Checklist for Task 1.2

- [X] Create `pkg/config/validation.go`
- [X] Create `pkg/config/errors.go`
- [X] Create `pkg/config/validation_test.go`
- [X] Implement `Validate()` method
- [X] Implement `ValidateWithContext()` method
- [X] All validation functions work correctly
- [X] `ValidationError` formats messages nicely
- [X] All test cases pass
- [X] Test coverage >85%
- [X] Run: `go test ./pkg/config -v`
- [X] Run: `go test ./pkg/config -cover`

---

## Testing Validation Manually

```bash
# Create a bad config
cat > bad-config.yaml <<EOF
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: "MyApp"  # Bad: uppercase
spec:
  imageName: myapp
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 0  # Bad: must be ≥1
  localPort: 70000  # Bad: too high
  servicePort: 8080
  env:
    - name: log-level  # Bad: must be uppercase
      value: info
EOF

# Parse and validate
go run ./cmd validate --config bad-config.yaml
# Should show all 4 errors at once!
```

---

## Next Steps

1. **Implement this task** ← You are here
2. Move to **Task 1.3** → Config loader uses Validate()
3. Move to **Task 1.4** → Context validator uses Spec.KubeContext
4. When loading config, call `ValidateWithContext()` with project root


