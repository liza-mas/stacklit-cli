# Stacklit

**108,000 lines of code. 4,000 tokens of index.**

One command makes any repo AI-agent-ready. No server, no setup.

[![License](https://img.shields.io/badge/license-MIT-green)](https://opensource.org/licenses/MIT)

## Fork notice

This repository is a fork of [glincker/stacklit](https://github.com/glincker/stacklit). This fork removes the MCP server/tooling surface and exposes the former MCP query tools as plain CLI commands instead:

| Former MCP tool | CLI command |
|-----------------|-------------|
| `find_module` | `stacklit find-module <query> -i stacklit.json` |
| `get_dependencies` | `stacklit get-dependencies <module> -i stacklit.json` |
| `get_hints` | `stacklit get-hints -i stacklit.json` |
| `get_hot_files` | `stacklit get-hot-files -i stacklit.json` |
| `get_module` | `stacklit get-module <name> -i stacklit.json` |

## Install and run

```bash
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | sh
stacklit generate-json -o stacklit.json
```

That is it. Builds and installs Stacklit from the main branch, then scans your codebase and generates the JSON index. The installer requires `git`, `go`, and `make`.

Other install options:

```bash
# Build from a branch with caller-provided Go and make
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | BRANCH=<branch> sh

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | INSTALL_DIR=<directory> sh

# Custom source repo
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | STACKLIT_SOURCE_REPO=<git-url> sh
```

From a local clone:

```bash
make install
stacklit --version
```

Use `INSTALL_DIR=<directory> make install` to install from a local clone into a custom directory.

## CI / GitHub Action

Use explicit workflow steps to install this fork and regenerate the JSON index:

```yaml
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
  with:
    go-version: '1.25'
- run: curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | sh
- run: stacklit generate-json -o stacklit.json
```

![Stacklit demo](demo.gif)

## What happens when you run it

```
$ stacklit generate-json -o stacklit.json
```

One file appears in your project:

| File | What it is | Commit it? |
|------|-----------|------------|
| `stacklit.json` | Codebase index for AI agents | **Yes** |

```bash
git add stacklit.json
git commit -m "add stacklit codebase index"
```

Done. Every AI agent that opens this repo can now read `stacklit.json` instead of scanning files.

## Why

AI coding agents burn most of their context window figuring out where things live. Reading one large file to find a function signature costs thousands of tokens. Five agents on the same repo each rebuild the same mental model from scratch.

**Without stacklit:** Agent reads 8-12 files. ~400,000 tokens. 45 seconds before writing a line.

**With stacklit:** Agent reads `stacklit.json`. ~4,000 tokens. Knows the structure instantly.

### Token efficiency (measured on real projects)

| Project | Language | Lines of code | Index tokens |
|---------|----------|---------------|-------------|
| Express.js | JavaScript | 21,346 | 3,765 |
| FastAPI | Python | 108,075 | 4,142 |
| Gin | Go | 23,829 | 3,361 |
| Axum | Rust | 43,997 | 14,371 |

See [examples/](https://github.com/glincker/stacklit/tree/master/examples) for full outputs.

## What is in stacklit.json

```json
{
  "modules": {
    "src/auth": {
      "purpose": "Authentication and session management",
      "files": 8, "lines": 1200,
      "exports": ["AuthProvider", "useSession()", "loginAction()"],
      "depends_on": ["src/db", "src/config"],
      "activity": "high"
    }
  },
  "hints": {
    "add_feature": "Create handler in src/api/, add route in src/index.ts",
    "test_command": "npm test"
  }
}
```

Modules, dependencies, exports with signatures, type definitions, git activity heatmap, framework detection, and hints for where to add features and how to run tests.

## Set up your AI tools

### Compact navigation map

```bash
stacklit derive         # print to stdout
```

Generates a ~250-token navigation map that replaces 3,000-8,000 tokens of agent exploration:

```
myapp | go | 14 modules | 8,420 lines
entry: cmd/api/main.go | test: go test ./...

modules:
  cmd/api/          entrypoint, routes, middleware
  internal/auth/    jwt, session | depends: store, config
  internal/store/   postgres | depended-by: auth, handler
```

### Manual setup

<details>
<summary>Configure manually instead</summary>

**Claude Code**  - add to `CLAUDE.md`:

```
Read stacklit.json before exploring files. Use modules to locate code, hints for conventions.
```

**Any other agent** - `stacklit.json` is a plain JSON file. Any tool that reads files can use it.

</details>

## Keep it updated

```bash
stacklit generate-json -o stacklit.json  # only update the JSON index
stacklit generate-json --workspace ..    # record the repo relative to a workspace root
stacklit generate-json --multi repos.txt # write a combined stacklit-multi.json
stacklit init-insights                   # create/update stacklit-insights.json
stacklit ai-summary                      # update insights with AI purposes, hints, and summary
stacklit derive --ai-summary             # include the stored AI summary in the compact map
stacklit diff              # check if the index is stale
```

`generate-json` automatically enriches the index from `stacklit-insights.json` when it exists. Use `--insights <file>` to read a different insights file; if that file is missing, Stacklit warns and continues without insights.

`stacklit diff` exits `0` when the index is current, `1` when sources changed, and `2` when the command cannot complete.

For a local agent cache, gitignore `stacklit.json` and refresh it from a local `post-commit` hook. See [USAGE.md](USAGE.md#optional-local-post-commit-refresh).

With `--multi`, `-o <file>` changes the multi-index output path from `stacklit-multi.json` to the given file.

<details>
<summary>GitHub Action for auto-updates</summary>

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

</details>

## Visual map

![Stacklit visual map](stacklit-og.png)

`stacklit view` opens the interactive HTML. Four views:

- **Graph** -- Force-directed dependency map. Click a node to see exports, types, files.
- **Tree** -- Collapsible directory hierarchy with file and line counts.
- **Table** -- Sortable module table with search filter.
- **Flow** -- Top-down dependency flow from entrypoints to leaves.

## 11 languages via tree-sitter

| Language | Extracts |
|----------|----------|
| Go | imports, exports with signatures, struct fields, interface methods |
| TypeScript/JS | imports (ESM, CJS, dynamic), classes, interfaces, type aliases |
| Python | imports, classes with methods, type hints, decorators |
| Rust | use/mod/crate, pub items with generics, trait methods |
| Java | imports, public classes, method signatures with types |
| C# | using directives, public types, method signatures |
| Ruby | require, classes, modules, methods |
| PHP | namespace use, classes, traits, public methods |
| Kotlin | imports, classes, objects, functions |
| Swift | imports, structs, classes, protocols |
| C/C++ | includes, functions, structs, typedefs |

Any other language gets basic support (line count + language detection).

## All CLI commands

```
stacklit generate-json -o stacklit.json  # generate only the JSON index
stacklit generate-json --insights stacklit-insights.json # enrich from insights
stacklit generate-json --workspace ..    # record the repo relative to a workspace root
stacklit generate-json --multi repos.txt # generate stacklit-multi.json from repo paths
stacklit init-insights -i stacklit.json -o stacklit-insights.json # seed curated insights
stacklit ai-summary -i stacklit.json -o stacklit-insights.json    # update AI-generated insights
stacklit find-module api -i stacklit.json  # search modules in an index
stacklit get-module internal/cli -i stacklit.json  # inspect one module
stacklit get-dependencies internal/cli -i stacklit.json  # module dependency edges
stacklit get-hints -i stacklit.json       # workflow hints
stacklit get-hot-files -i stacklit.json   # git churn hotspots
stacklit view -i stacklit.json   # regenerate HTML from an index, open in browser
stacklit diff -i stacklit.json   # check if an index is stale
stacklit derive -i stacklit.json # print compact nav map (~250 tokens)
stacklit derive --ai-summary -i stacklit.json # include stored AI summary
```

<details>
<summary>Configuration (.stacklitrc.json)</summary>

```json
{
  "ignore": ["vendor/", "generated/"],
  "max_depth": 3,
  "output": {
    "json": "stacklit.json"
  }
}
```

`generate-json` and `diff` honor `output.json` when no explicit output/input flag is passed. Legacy `output.mermaid` and `output.html` keys may exist in older config files, but this CLI no longer exposes the old full-generation command that wrote `DEPENDENCIES.md` and configured HTML. `view` always writes `stacklit.html`.

`diff` uses differentiated exit codes: `0` means the index is current, `1` means sources changed, and `2` means the command failed.

</details>

<details>
<summary>Curated insights (stacklit-insights.json)</summary>

`stacklit-insights.json` stores stable semantic knowledge that should enrich the generated index: module purposes, workflow hints, and architecture summaries.

```json
{
  "purpose": {
    "internal/cli": "CLI commands and wiring",
    "engine": "Index generation pipeline"
  },
  "hints": {
    "add_feature": "Add commands in internal/cli and register them in root.go",
    "test_command": "go test ./..."
  },
  "architecture": {
    "ai_summary": "..."
  }
}
```

Create or refresh it with `stacklit init-insights`. Existing entries are preserved; pass `--prune` to remove purpose entries for modules that no longer exist. If `stacklit.json` is missing, `init-insights` scans in memory and writes only the insights file.

Generate AI insights separately with `stacklit ai-summary`. By default it runs `claude -p --system-prompt "<insight-generation instructions>"`, sends the compact JSON snapshot on stdin, and expects stdout to be valid `stacklit-insights.json` content containing `purpose`, `hints`, and `architecture.ai_summary`. Configure another local summary command prefix with `STACKLIT_SUMMARY_CMD`, and adjust the timeout with `STACKLIT_SUMMARY_TIMEOUT`. Stacklit appends the generated prompt after the prefix, so custom commands receive instructions as their final argument and the JSON snapshot on stdin. `STACKLIT_SUMMARY_CMD` is split on whitespace, so use a wrapper script for complex quoted arguments.

Use `stacklit derive --ai-summary` to include the stored summary in the compact map. This is read-only and does not call the summary command.

</details>

## How it compares

| Tool | Approach | Tokens | Committable | Visual map |
|------|----------|--------|-------------|------------|
| **Stacklit** | Structured index | ~250 | Yes | Yes |
| Repomix | Full dump | 50k-500k | No | No |
| code2prompt | Full dump | 50k-500k | No | No |
| Aider repo-map | Tree-sitter + PageRank | ~1k | No | No |

[Full comparison with 7 tools →](https://github.com/glincker/stacklit/discussions/13)

## Monorepo support

Auto-detects: pnpm, npm, yarn workspaces, Go workspaces, Turborepo, Nx, Lerna, Cargo workspaces, and convention directories (`apps/`, `packages/`, `services/`).

## How does Stacklit compare to Repomix?

Repomix concatenates all files into one prompt (50k-500k tokens). Stacklit parses code structure and generates a ~250-token navigation map. Use Repomix for small repos and one-shot chats. Use Stacklit for daily AI-assisted development on larger codebases. See the [full comparison](https://github.com/glincker/stacklit/discussions/13).

## FAQ

**Does Stacklit read my code?**
Yes, locally. It parses source files with tree-sitter to extract structure (imports, exports, types). No code is sent anywhere unless you run `stacklit ai-summary`, which invokes the configured summary CLI.

**What if my language isn't supported?**
Stacklit falls back to basic support (line count + language detection) for any language not in the tree-sitter list. The module map, dependency graph, and git activity still work.

**Can I use Stacklit with GitHub Copilot?**
Yes. Commit `stacklit.json` and reference it in your Copilot instructions.

## Documentation

- [USAGE.md](USAGE.md) -- full usage guide, command reference, and configuration
- [COMPARISON.md](COMPARISON.md) -- head-to-head comparison with Repomix and code2prompt
- [SKILL.md](SKILL.md) -- instructions for AI agents on how to use stacklit.json

## Contributing

```bash
make build   # build binary
make test    # run all tests
```

Contributions welcome. See [open issues](https://github.com/glincker/stacklit/issues) or start a [discussion](https://github.com/glincker/stacklit/discussions).

## License

MIT
