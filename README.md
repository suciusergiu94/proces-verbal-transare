# proces-verbal-transare

A [Wails v2](https://wails.io) desktop app (Go backend + Vite/TypeScript frontend).

## Prerequisites

This project was scaffolded without network access to the Go module proxy, so
dependencies haven't been fetched or verified yet. On your machine, with
normal internet access, install:

- Go 1.21+: https://go.dev/dl/
- Node.js 18+ (already present on this machine)
- The Wails CLI:

  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

- Platform build tools for macOS: Xcode Command Line Tools (`xcode-select --install`)

Then verify everything is in order:

```bash
wails doctor
```

## First-time setup

```bash
go mod tidy        # fetches Go dependencies and generates go.sum
cd frontend
npm install         # fetches frontend dependencies
cd ..
```

## Development

```bash
wails dev
```

This starts the app with hot reload for the frontend, and regenerates the
TypeScript bindings in `frontend/wailsjs/` from the `App` struct's Go methods
(the versions checked in here are hand-written placeholders just so the
frontend type-checks before your first `wails dev` run).

## Production build

```bash
wails build
```

The binary is written to `build/bin/`.
