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

Installer environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `BRANCH` | `main` | Git branch to clone and build |
| `INSTALL_DIR` | `$HOME/.local/bin` | Directory for the installed `stacklit` binary |
| `STACKLIT_REPO_OWNER` | `liza-mas` | GitHub owner used to build the default source URL |
| `STACKLIT_REPO_NAME` | `stacklit-cli` | GitHub repo used to build the default source URL |
| `STACKLIT_SOURCE_REPO` | `https://github.com/$STACKLIT_REPO_OWNER/$STACKLIT_REPO_NAME.git` | Full Git URL to clone |
| `STACKLIT_SOURCE_TMPDIR` | `$TMPDIR` or `/tmp` | Parent directory for the temporary source checkout |

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

`init-insights` creates `stacklit-insights.json` from the current index. `ai-summary` refreshes AI-generated module purposes, workflow hints, and the architecture summary in that insights file. The final `generate-json` rebuilds `stacklit.json` enriched with the curated insights.

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

Refresh AI-generated insights:

```bash
stacklit generate-json
stacklit ai-summary
stacklit generate-json
```

Include the stored AI summary in the compact navigation map:

```bash
stacklit derive --ai-summary
```

### Optional local post-commit refresh

If `stacklit.json` is a local agent cache, add it to `.gitignore` and refresh it after each commit instead of committing index updates:

```gitignore
stacklit.json
stacklit.html
```

Create `index.sh` in the repo root:

```bash
#!/usr/bin/env bash
set -euo pipefail

if command -v stacklit >/dev/null; then
  code=0
  if [ -f stacklit.json ]; then
    stacklit diff >/dev/null || code=$?
    case "$code" in
      0) [ "${1:-}" = "ai" ] || exit 0 ;;
      1) ;;
      *) echo "stacklit diff failed"; exit "$code" ;;
    esac
  fi

  echo "Stacklit Indexing..."
  stacklit generate-json
  stacklit init-insights
  if [ "${1:-}" = "ai" ]; then
    echo "Adding AI-generated insights..."
    stacklit ai-summary
  fi
  stacklit generate-json
  echo "Wrote stacklit.json"
fi
```

Install a local hook at `.git/hooks/post-commit`:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$repo_root" ]; then
  exit 0
fi

case "$repo_root" in
  */.worktrees/*)
    exit 0
    ;;
esac

if [ ! -x "$repo_root/index.sh" ]; then
  exit 0
fi

cd "$repo_root"
./index.sh
```

Then make both scripts executable:

```bash
chmod +x index.sh .git/hooks/post-commit
```

This setup keeps `stacklit.json` fresh for local agents without creating generated-file churn in commits. Track `stacklit-insights.json` separately if you want to share curated module purposes, hints, or AI summaries.

If your team wants `stacklit.json` committed, do not use this post-commit pattern as the only refresh step. Regenerate the index before committing or in CI so the commit contains the updated file.

### Regenerate after making changes

```bash
stacklit generate-json -o stacklit.json
```

`generate-json` automatically enriches the index from `stacklit-insights.json` when that file exists. Use `--insights <file>` to read a different insights file; if that file is missing, Stacklit warns and continues without insights.

With `--multi`, `generate-json` reads a plain text file of repo paths and writes a combined multi-repo index. The default output is `stacklit-multi.json`; pass `-o <file>` to choose a different multi-index path.

### Curate insights

```bash
stacklit init-insights
stacklit ai-summary
stacklit generate-json
```

`init-insights` creates or updates `stacklit-insights.json` with module purposes and hints while preserving existing edits. `ai-summary` asks the configured local agent to generate insights JSON, then updates module purposes, workflow hints, and `architecture.ai_summary` in the insights file.

If `stacklit.json` is missing, `init-insights` runs the indexer in memory and creates `stacklit-insights.json` without writing `stacklit.json`. By default it preserves purpose entries for modules that no longer exist; use `--prune` to remove those stale purpose entries.

### Configure AI summaries

`ai-summary` reads `stacklit.json`, sends a compact architecture snapshot to a local summary command on stdin, and expects stdout to be valid `stacklit-insights.json` content containing `purpose`, `hints`, and `architecture.ai_summary`. Non-empty generated values are merged into the output insights file.

`derive --ai-summary` is read-only. It includes the existing `architecture.ai_summary` from `stacklit.json` and does not invoke the local summary command. If the summary is missing, refresh it with `stacklit ai-summary` and then rebuild `stacklit.json` with `stacklit generate-json`.

By default, Stacklit runs:

```bash
claude -p --system-prompt "<insight-generation instructions>"
```

Override the command prefix with `STACKLIT_SUMMARY_CMD`. Stacklit appends the insight-generation prompt after this prefix and sends the compact JSON snapshot on stdin:

```bash
STACKLIT_SUMMARY_CMD="claude -p --system-prompt" stacklit ai-summary
STACKLIT_SUMMARY_CMD="codex exec --dangerously-bypass-approvals-and-sandbox" stacklit ai-summary
```

For the default Claude command, insight-generation instructions are passed with `--system-prompt` and stdin contains only the compact JSON snapshot. Custom `STACKLIT_SUMMARY_CMD` prefixes receive the generated prompt as their final argument and must write the insights JSON object to stdout.

`STACKLIT_SUMMARY_CMD` is split on whitespace. Shell-style quoted arguments are not preserved, so prefer simple command lines or a small wrapper script for complex invocations.

The timeout defaults to 120 seconds. Override it with `STACKLIT_SUMMARY_TIMEOUT`:

```bash
STACKLIT_SUMMARY_TIMEOUT=300 stacklit ai-summary
```

`ai-summary` uses this local CLI path; it does not call the direct Anthropic API helper.

### Check if the index is stale

```bash
stacklit diff
```

Prints "Index is up to date" or tells you what changed.

Exit codes: `0` means the index is current, `1` means sources changed, and `2` means the command failed.

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
| `stacklit ai-summary -i stacklit.json -o stacklit-insights.json` | Update AI-generated purposes, hints, and summary in the insights file |
| `stacklit derive --ai-summary -i stacklit.json` | Print the compact map with the stored AI summary included |
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
    "json": "stacklit.json"
  }
}
```

Keys: `ignore` (extra paths on top of `.gitignore`), `max_depth` (module detection depth, default 4), `max_modules` (collapse threshold, default 200), `max_exports` (per module, default 10), and `output.json` (default JSON output path).

`stacklit generate-json` and `stacklit diff` honor `output.json` when `-o` or `-i` is not passed. The legacy config keys `output.mermaid` and `output.html` may still be present in old config files, but this CLI no longer has the old full-generation command that wrote `DEPENDENCIES.md` and configured HTML. `stacklit view` always regenerates `stacklit.html` from an index.

---

## Reading stacklit.json

### Project info

```json
{
  "project": { "name": "my-app", "type": "monorepo" },
  "tech": {
    "primary_language": "typescript",
    "frameworks": ["React", "Express"],
    "framework_patterns": [
      {
        "name": "Express",
        "routes": "routes/",
        "entry": "server.js"
      }
    ]
  }
}
```

**framework_patterns** -- Optional framework-specific structure detected from config files, dependencies, and conventional directories. Fields can include `config_files`, `routes`, `api`, `middleware`, `models`, and `entry`.

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

### Query commands

`find-module <query>` searches both module names and module purposes. Results are sorted by module name and capped at five matches so the output stays compact.

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
