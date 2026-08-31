# Credora Decisioning Workspace

A developer and analyst-facing interface for interacting with the Credora credit decisioning engine.

## Running Locally

```bash
# From the workspace directory
cd apps/workspace
npm install
npm run dev
```

The workspace runs on http://localhost:3000 and proxies API calls to the Credora Engine at http://localhost:8080.

## Prerequisites

The Credora Engine must be running:

```bash
cd services/engine
go run ./cmd/server
```

## Stack

- **Preact** (~3KB gzipped) — React-like UI at minimal size
- **HTM** — JSX-like syntax without build complexity
- **Vite** — Fast dev server and build tool
- **Vitest** — Test runner

No UI component libraries. Pure CSS.

## Architecture

```
Workspace (Preact)
    ↓ HTTP (fetch)
Credora Engine API (Go)
    ↓
Assessment Service
    ↓
Policy Evaluator
    ↓
Decision + Evidence
```

The workspace is a pure API consumer. No decisioning logic exists in the frontend.

## Routes

| Route | Purpose |
|---|---|
| `#/` | Workspace home |
| `#/policies` | View available policies and versions |
| `#/assessments` | Look up assessments by ID |
| `#/assessments/new` | Create and run a new assessment |
| `#/assessments/:id` | View assessment detail with decision and evidence |

## API Configuration

The engine API URL defaults to the Vite proxy (`http://localhost:8080`). To use a different URL, set `VITE_API_URL` in a `.env` file:

```
VITE_API_URL=http://localhost:8080
```

## Tests

```bash
npm test
```

## Build

```bash
npm run build
```

Output: `dist/` directory.
