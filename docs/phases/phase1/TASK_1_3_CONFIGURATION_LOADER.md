# Task 1.3: Implement Configuration Loader

## Overview

This task implements the **configuration discovery and loading system**. It must:
1. **Discover** `.kudev.yaml` in project hierarchy
2. **Load** YAML and parse to Go types
3. **Validate** using validators from Task 1.2
4. **Apply defaults** for optional fields
5. **Provide clear errors** if config not found

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (file system traversal, YAML parsing)  
**Dependencies**: Task 1.1 (Types), Task 1.2 (Validation)  
**Files to Create**:
- `pkg/config/loader.go` — ConfigLoader interface + implementation
- `pkg/config/defaults.go` — Default values
- `pkg/config/loader_test.go` — Tests

---

## The Problem Config Loading Solves

Users should be able to run `kudev` from anywhere in their project:

```bash
$ cd ~/project/src/components
$ kudev status
# Should find ~/project/.kudev.yaml automatically, not fail

$ cd ~/project
$ kudev status  
# Should find ./.kudev.yaml

$ cd ~/project && kudev --config ./config/dev.yaml status
# Should respect --config override
```

---

## Discovery Algorithm

The **discovery algorithm** searches for `.kudev.yaml`:

```
1. Check for --config flag
   └─ If set: Use that path (must exist)
   
2. Check current directory
   └─ If found: Use it
   
3. Check parent directories (walk up to root)
   └─ Stop at project root (heuristics: .git, go.mod, .kudev.yaml)
   └─ If found: Use it
   
4. Check home directory (~/.kudev/config)
   └─ For global config (not recommended for projects)
   
5. Not found
   └─ Return helpful error with what we searched
```

### Project Root Detection

Heuristics to detect project root:
1. Directory containing `.git` (VCS root)
2. Directory containing `go.mod` (Go project root)
3. Directory containing `.kudev.yaml` (kudev project root)
4. Directory containing `package.json` (Node project root)
5. Current working directory + parent == root `/` or `C:\` (filesystem root)

---

---

## Key Design Decisions

### Decision 1: What to Default

**Question**: Should we default ALL fields?

**Answer**: **Default cautiously**
- ✅ Default: namespace (standard), replicas (safe), ports (common)
- ❌ Don't default: metadata.name (must be explicit), imageName, dockerfilePath

**Reasoning**: 
- Required fields prevent silent failures
- Optional fields benefit from smart defaults

### Decision 2: Search Order

**Question**: Search from CWD upward or downward?

**Answer**: **Search upward from CWD**
```bash
/project/
  .kudev.yaml
  src/
    components/
      mycomponent/
        $ kudev status  # Run from here
        # Finds /project/.kudev.yaml by walking up
```

This matches: `kubectl`, `git`, `node` package.json discovery

### Decision 3: Project Root Detection

**Question**: What heuristics to use?

**Answer**: **Multiple heuristics** (.git, go.mod, package.json, etc.)
- Works with many project types
- Stops search at first found marker
- Prevents infinite loop on symlinks

---

## Critical Points

### 1. Prevent Infinite Loops on Symlinks

```go
visited := make(map[string]bool)

for {
    if visited[current] {
        break  // Already saw this path
    }
    visited[current] = true
    
    // ... search logic ...
    
    parent := filepath.Dir(current)
    if parent == current {
        break  // Reached filesystem root
    }
    current = parent
}
```

### 2. Error Messages Must Be Helpful

❌ Bad:
```
config not found
```

✅ Good:
```
configuration file (.kudev.yaml) not found

Searched in:
  - /home/user/project/src/components
  - /home/user/project/src
  - /home/user/project
  - /home/user
  - /home
  - /

Suggestions:
  - Run 'kudev init' to create a new .kudev.yaml
  - Or place .kudev.yaml in your project root
  - Or specify config path with: kudev --config <path>
```

### 3. Relative Path Resolution

Two contexts where paths are relative:
1. **In YAML**: `dockerfilePath: ./Dockerfile`
   - Should be relative to project root (not CWD)
   - Resolved in loader.go

2. **--config flag**: `kudev --config ./dev.yaml`
   - Should be relative to CWD (user perspective)
   - Resolved with `filepath.Join(WorkingDir, path)`

---

## Checklist for Task 1.3

- [X] Create `pkg/config/loader.go`
- [X] Create `pkg/config/defaults.go`
- [X] Create `pkg/config/loader_test.go`
- [X] Implement `FileConfigLoader` type
- [X] Implement `Load()` method (discovery + loading)
- [X] Implement `LoadFromPath()` method
- [X] Implement `Save()` method
- [X] Implement `discover()` algorithm
- [X] Implement project root detection
- [X] Apply defaults before validation
- [X] Generate helpful error messages
- [X] All tests pass: `go test ./pkg/config -v`
- [X] Test coverage >80%

---

## Testing the Loader Manually

```bash
# Create test config
mkdir -p /tmp/kudev-test/src
cat > /tmp/kudev-test/.kudev.yaml <<EOF
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: test-app
spec:
  imageName: test-app
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 1
  localPort: 8080
  servicePort: 8080
EOF

# Create Dockerfile
touch /tmp/kudev-test/Dockerfile

# Test discovery from subdirectory
cd /tmp/kudev-test/src
kudev validate
# Should find /tmp/kudev-test/.kudev.yaml

# Test explicit config
cd /tmp
kudev --config /tmp/kudev-test/.kudev.yaml validate
# Should work
```

---

## Integration with Other Tasks

```
Task 1.1 (Types)
    ↓
Task 1.2 (Validation)
    ↓
Task 1.3 (Loader) ← You are here
    ↓
Load → Validate → Ready for Task 1.4 (Context)
```

Task 1.3 is the **glue** that ties together:
- File system (discovery)
- YAML parsing (loading)
- Validation (checking)
- Defaults (user experience)

---

## Next Steps

1. **Implement this task** ← You are here
2. **Task 1.4** → Use loader to get config, validate context
3. **Task 1.5** → Cobra commands use loader to get config
4. **Phase 2** → Extend loader to support ConfigMaps, Secrets


