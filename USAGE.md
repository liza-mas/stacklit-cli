# Stacklit usage guide

## First time setup

### 1. Install

Pick one:

```bash
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | sh
```

Options:

```bash
# Build from a branch with caller-provided Go and make
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | BRANCH=<branch> sh

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | INSTALL_DIR=<directory> sh
```

From a local clone:

```bash
make install
stacklit --version
```

Use `INSTALL_DIR=<directory> make install` to install from a local clone into a custom directory.

The installer builds from source and requires `git`, `go`, and `make`.

### 2. Generate your index

```bash
cd your-project
stacklit generate-json -o stacklit.json
```

### 3. Optional: create curated insights

```bash
stacklit init-insights
stacklit ai-summary
stacklit generate-json
```

`init-insights` creates `stacklit-insights.json` from the current index. `ai-summary` adds an AI-generated architecture summary to that insights file. The final `generate-json` rebuilds `stacklit.json` enriched with the curated insights.

Skip `ai-summary` if you do not want AI-generated summaries.

### 4. Commit the index

```bash
git add stacklit.json stacklit-insights.json
git commit -m "add stacklit codebase index"
git push
```

If you skipped insights, only add `stacklit.json`.

`stacklit.html` is gitignored. It regenerates locally with `stacklit view`.

### 5. Tell your AI tool about it

**Claude Code** -- add to `CLAUDE.md`:
```
Read stacklit.json before exploring files. Use modules to locate code, hints for conventions.
```

**Copilot** -- add to `.github/copilot-instructions.md`:
```
Read stacklit.json first to understand codebase structure before exploring files.
```

**Any other agent** -- `stacklit.json` is a plain JSON file. Point the agent at it.

---

## Daily use

### Typical workflow

First-time setup with insights:

```bash
stacklit generate-json
stacklit init-insights
stacklit ai-summary
stacklit generate-json
git add stacklit.json stacklit-insights.json
git commit -m "add stacklit index"
```

Daily use after code changes:

```bash
stacklit generate-json
stacklit diff
```

Refresh curated purposes and hints after module changes:

```bash
stacklit init-insights
stacklit generate-json
```

Refresh the AI architecture summary:

```bash
stacklit generate-json
stacklit ai-summary
stacklit generate-json
```

### Regenerate after making changes

```bash
stacklit generate-json -o stacklit.json
```

`generate-json` automatically enriches the index from `stacklit-insights.json` when that file exists. Use `--insights <file>` to read a different insights file; if that file is missing, Stacklit warns and continues without insights.

### Curate insights

```bash
stacklit init-insights
stacklit ai-summary
stacklit generate-json
```

`init-insights` creates or updates `stacklit-insights.json` with module purposes and hints while preserving existing edits. `ai-summary` updates `architecture.ai_summary` in the insights file.

### Check if the index is stale

```bash
stacklit diff
```

Prints "Index is up to date" or tells you what changed.

### Open the visual map

```bash
stacklit view
```

Regenerates `stacklit.html` and opens it in your browser.

---

## Command reference

| Command | What it does |
|---------|-------------|
| `stacklit generate-json -o stacklit.json` | Generate only the JSON index, quietly |
| `stacklit generate-json --insights stacklit-insights.json` | Enrich the generated index from an insights file |
| `stacklit generate-json --workspace ..` | Record the repo root relative to a workspace root |
| `stacklit generate-json --multi repos.txt` | Generate a combined `stacklit-multi.json` from repo paths |
| `stacklit init-insights -i stacklit.json -o stacklit-insights.json` | Seed or update curated insights |
| `stacklit ai-summary -i stacklit.json -o stacklit-insights.json` | Update the AI summary in the insights file |
| `stacklit find-module api -i stacklit.json` | Search modules in an index |
| `stacklit get-module internal/cli -i stacklit.json` | Get full info for one module |
| `stacklit get-dependencies internal/cli -i stacklit.json` | Get dependency edges for a module |
| `stacklit get-hints -i stacklit.json` | Get workflow hints |
| `stacklit get-hot-files -i stacklit.json` | Get git churn hotspots |
| `stacklit view -i stacklit.json` | Regenerate HTML from an index and open in browser |
| `stacklit diff -i stacklit.json` | Check if an index is stale |
| `stacklit derive -i stacklit.json` | Print compact navigation map (~250 tokens) to stdout |
| `stacklit --version` | Print version |

---

## Configuration

Create `.stacklitrc.json` in your project root (optional):

```json
{
  "ignore": ["vendor/", "generated/", "*.pb.go"],
  "max_depth": 3,
  "max_modules": 150,
  "max_exports": 15,
  "output": {
    "json": "stacklit.json",
    "mermaid": "DEPENDENCIES.md",
    "html": "stacklit.html"
  }
}
```

Keys: `ignore` (extra paths on top of `.gitignore`), `max_depth` (module detection depth, default 4), `max_modules` (collapse threshold, default 200), `max_exports` (per module, default 10), and `output` (override generated file names).

---

## Reading stacklit.json

### Project info

```json
{
  "project": { "name": "my-app", "type": "monorepo" },
  "tech": {
    "primary_language": "typescript",
    "frameworks": ["React", "Express"]
  }
}
```

### Modules

Each module represents a directory of related source files:

```json
"src/auth": {
  "purpose": "Authentication and session management",
  "language": "typescript",
  "files": 8,
  "lines": 1200,
  "file_list": ["service.ts", "middleware.ts", "types.ts"],
  "exports": ["AuthProvider", "useSession()", "loginAction(email, password)"],
  "type_defs": {
    "AuthState": "user User, token string, isLoading boolean"
  },
  "depends_on": ["src/db", "src/config"],
  "depended_by": ["src/api", "src/middleware"],
  "activity": "high"
}
```

**purpose** -- What the module does (auto-generated from directory name and contents).

**exports** -- Public functions, classes, and types with signatures. For Go, includes full parameter and return types.

**type_defs** -- Struct/interface/class field definitions. Lets agents understand data shapes without reading source.

**depends_on / depended_by** -- Module-level dependency graph. Tells you what breaks if you change this module.

**activity** -- high/medium/low based on 90-day git commit frequency.

### Dependencies

```json
"dependencies": {
  "edges": [["src/api", "src/auth"], ["src/auth", "src/db"]],
  "most_depended": ["src/db", "src/config"],
  "isolated": ["scripts"]
}
```

**most_depended** -- Modules with the most dependents. Change these carefully.

**isolated** -- Modules with no incoming or outgoing dependencies.

### Hints

```json
"hints": {
  "add_feature": "Create handler in src/api/, add route in src/routes/index.ts",
  "test_command": "npm test",
  "env_vars": ["DATABASE_URL", "JWT_SECRET"]
}
```

Agent-actionable instructions generated from codebase analysis.

---

## Monorepo support

Stacklit auto-detects workspaces:

| Tool | Detection |
|------|-----------|
| pnpm | `pnpm-workspace.yaml` |
| npm/yarn | `package.json` workspaces field |
| Go | `go.work` |
| Turborepo | `turbo.json` |
| Nx | `nx.json` |
| Lerna | `lerna.json` |
| Cargo | `Cargo.toml` workspace section |
| Convention | `apps/`, `packages/`, `services/`, `libs/` directories |

Each workspace appears as a group of modules in the index.

---

## CI/CD

### GitHub Action

```yaml
name: Update stacklit index
on:
  push:
    branches: [main]
    paths-ignore: ['stacklit.json', '**.md']

jobs:
  stacklit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | sh
      - run: stacklit generate-json
      - uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "chore: update stacklit index"
          file_pattern: "stacklit.json"
```
