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

### 3. Commit the index

```bash
git add stacklit.json
git commit -m "add stacklit codebase index"
git push
```

`stacklit.html` is gitignored. It regenerates locally with `stacklit view`.

### 4. Tell your AI tool about it

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

### Regenerate after making changes

```bash
stacklit generate-json -o stacklit.json
```

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
