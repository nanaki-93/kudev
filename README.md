# Kudev - K8s Helper

A CLI tool to streamline local Kubernetes development with automatic building, deploying, and hot-reloading.

## Overview

K8s Helper simplifies the development workflow by automating Docker image builds, Kubernetes deployments, and providing real-time feedback through logs and port forwarding.

## Table of Contents

- [Phase 1: Project Scaffolding & CLI Design](#phase-1-project-scaffolding--cli-design)
- [Phase 2: Local Image Orchestration](#phase-2-local-image-orchestration)
- [Phase 3: Kubernetes Logic](#phase-3-kubernetes-logic-client-go)
- [Phase 4: Feedback Loop & Port Forwarding](#phase-4-feedback-loop--port-forwarding)
- [Phase 5: Live Watcher](#phase-5-live-watcher-hot-reload)

---

## Phase 1: Project Scaffolding & CLI Design

Build the CLI interface using the [Cobra](https://github.com/spf13/cobra) library for command-line argument handling and the [Viper](https://github.com/spf13/viper) library for configuration management.

### Setup

```bash
go mod init github.com/yourname/kudev
```

### Commands

- **`kudev up`** — The main entry point to build and deploy your application
- **`kudev status`** — Check if the local project is healthy in the cluster

### Configuration

Create a `helper.yaml` file in your project root:

```yaml
app_name: myapp
namespace: default
dockerfile_path: ./Dockerfile
```

---

## Phase 2: Local Image Orchestration

Your tool needs to package the local code. Choose one of two approaches:

| Approach | Library | Docker Daemon Required |
|----------|---------|------------------------|
| **Docker SDK** | Official Go Docker SDK | Yes |
| **Daemonless** | Google `ko` or `kaniko` | No |

### Image Tagging Strategy

Dynamically tag images with timestamps to ensure Kubernetes pulls the latest version:

```
myapp:local-TIMESTAMP
```

Example: `myapp:local-20250115-143025`

---

## Phase 3: Kubernetes Logic (Client-Go)

Interact with your cluster using [client-go](https://github.com/kubernetes/client-go), the official Kubernetes client library.

### Key Implementation Steps

1. **Authentication**
   - Load the user's `~/.kube/config` credentials
   - Set up client connection to the cluster

2. **Manifest Templating**
   - Use Go's `text/template` to inject your new image tag dynamically
   - Avoid static manifest files

3. **Cluster Operations**
   - Check if a Namespace exists (create it if not)
   - Update the Deployment with the new image
   - Ensure a Service (and optional Ingress) is present to expose the app

---

## Phase 4: Feedback Loop & Port Forwarding

A helper tool is only useful if it provides real-time feedback to the developer.

### Features

- **Log Streaming**
  - Use client-go to tail the logs of newly created pods
  - Stream logs to the user's terminal in real-time

- **Automatic Port Forwarding**
  - Implement a port-forwarding routine in Go
  - Make the app automatically accessible at `localhost:8080`

---

## Phase 5: Live Watcher (Hot Reload)

Enable automatic redeployment on file changes for a seamless development experience.

### Implementation

1. **File Watching**
   - Integrate the [fsnotify](https://github.com/fsnotify/fsnotify) library
   - Watch for `.go`, `.yaml`, and `.html` file changes

2. **Trigger Re-deployment**
   - On file change, automatically trigger Phase 2 (Build) and Phase 3 (Deploy)
   - Provide visual feedback to the user on re-deployment status

---

## Getting Started

```bash
# Clone the repository
git clone github.com/yourname/kudev

# Navigate to project
cd kudev

# Install dependencies
go mod download

# Build the CLI
go build -o kudev ./cmd/main.go

# Deploy your app
./kudev up
```

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MIT
