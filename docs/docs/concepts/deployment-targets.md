---
sidebar_position: 2
---

# Deployment Targets

vibeD supports three deployment targets. It auto-detects which are available and picks the best one.

## Knative Serving (Preferred)

Knative provides the best experience for web artifacts:

- **Automatic HTTPS** with auto-generated certificates
- **Scale-to-zero** for cost efficiency
- **Clean URLs** like `my-app.default.example.com`
- **Revision-based rollbacks**

vibeD creates Knative `Service` resources that manage revisions, routing, and scaling automatically.

## Kubernetes (Always Available)

Plain Kubernetes deployments as a fallback:

- **Deployment + Service** with NodePort
- **Always available** on any Kubernetes cluster
- **Manual scaling** via replica count

vibeD creates a `Deployment` and a `Service` with `NodePort` type.

## wasmCloud (WebAssembly)

For artifacts compiled to WebAssembly components:

- **OAM Application** manifests via the wasmcloud-operator
- **Lightweight** and fast cold starts
- **Distributed** across wasmCloud hosts
- **Wasm build pipeline** — vibeD compiles Go (TinyGo) and Rust source to wasm components via `wash build`

### Prerequisites

- [wasmcloud-operator](https://github.com/wasmCloud/wasmcloud-operator) installed in the cluster
- NATS server (required by wasmCloud)
- wadm (wasmCloud Application Deployment Manager)
- An OCI registry to store wasm component artifacts

### Supported Languages

| Language | Toolchain | Notes |
|----------|-----------|-------|
| Go | TinyGo | Compiled with `-target=wasi` |
| Rust | cargo-component | Native wasm component support |

### How It Works

1. vibeD scaffolds `wasmcloud.toml` and WIT interface files alongside your source code
2. A Kubernetes Job runs `wash build` to compile the code to a wasm component
3. The built component is pushed to the OCI registry via `wash push`
4. vibeD creates a wadm OAM Application manifest with an HTTP server capability provider
5. The wasmcloud-operator deploys the component to wasmCloud hosts

### Static Files

wasmCloud is designed for application logic, not static file serving. When you deploy static HTML/CSS/JS files with `target=wasmcloud`, vibeD automatically falls back to Knative or Kubernetes.

### Configuration

```yaml
config:
  wasmcloud:
    latticeId: "default"          # wasmCloud lattice ID
    builder:
      image: "ghcr.io/vibed/wasm-builder:latest"  # Builder image with wash + toolchains
      timeout: "10m"
      insecure: false             # Set true for non-TLS registries
```

Environment variable overrides: `VIBED_WASMCLOUD_LATTICE_ID`, `VIBED_WASMCLOUD_BUILDER_IMAGE`, `VIBED_WASMCLOUD_BUILDER_INSECURE`.

## Target Selection

When `target` is set to `auto` (default), vibeD picks the target in this priority:

1. **Knative** - If `serving.knative.dev` CRDs exist
2. **wasmCloud** - If `core.oam.dev` CRDs exist
3. **Kubernetes** - Always available as fallback

You can override this per-artifact by passing `target` to the `deploy_artifact` MCP tool.
