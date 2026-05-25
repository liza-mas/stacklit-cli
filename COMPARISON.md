# AI Codebase Context Tools - Comparison

> How do you give AI agents codebase context? Here's every approach compared.
>
> Last updated: 2026-04-10 | [Submit corrections](https://github.com/glincker/stacklit/issues)

## The Problem

AI coding agents (Claude Code, Cursor, Copilot, Aider) need to understand your codebase before they can help. Without context, they waste thousands of tokens exploring  - reading files, grepping, globbing  - just to figure out your project structure.

Different tools solve this differently. Some dump everything. Some build knowledge graphs. Some generate compressed maps. Here's how they compare.

## Quick Comparison

| Tool | Stars | Approach | Output | Token Cost* | Languages | Dependencies | Standalone |
|------|-------|----------|--------|-------------|-----------|-------------|-----------|
| [Repomix](https://github.com/yamadashy/repomix) | 23k | Full file dump | XML/MD/JSON | 50k-500k | Any | No | CLI (Node) |
| [Gitingest](https://github.com/coderamp-labs/gitingest) | 14k | Full file dump | Text | 50k-500k | Any | No | Web + CLI |
| [code2prompt](https://github.com/mufeedvh/code2prompt) | 7k | Full dump + templates | Text | 50k-500k | Any | No | CLI (Rust) |
| [files-to-prompt](https://github.com/simonw/files-to-prompt) | 2.6k | Concat files | XML | 50k-500k | Any | No | CLI (Python) |
| [Aider repo-map](https://github.com/Aider-AI/aider) | 43k** | Tree-sitter + PageRank | Text | ~1k | 40+ | Yes | Locked to Aider |
| **[Stacklit](https://github.com/glincker/stacklit)** | -- | Tree-sitter + module map | JSON + HTML | **~250 (compact map)** | 11 | Yes | CLI (Go) |

*Token cost = tokens consumed to give an agent full project context on a ~10k-line repo.
**Aider's 43k stars are for the entire tool, not just the repo-map feature.

## Approach Breakdown

### Full-Content Dumpers
**Repomix, Gitingest, code2prompt, files-to-prompt**

Concatenate all source files into one big prompt. Simple, works everywhere, but:
- Burns 50k-500k tokens on medium repos
- Often exceeds context windows entirely
- No structural intelligence  - agent still has to parse everything
- Repomix's `--compress` mode uses tree-sitter to reduce output, but remains per-file (no cross-file dependency analysis)

Best for: Small repos (<5k lines), one-shot conversations, pasting into ChatGPT.

### Structural Index (Stacklit)

Parses code with tree-sitter, builds a module-level dependency graph, outputs a compact navigation map:
- **~250 tokens** for the compact map (vs 50k-500k for dumpers)
- Static artifact  - commit `stacklit.json` to your repo
- Self-contained HTML visualization
- No running server needed

Best for: Any repo, any AI tool, zero ongoing maintenance.

### IDE-Integrated (Proprietary)
**Cursor, Continue.dev, GitHub Copilot, Sourcegraph Cody**

Built into IDEs, not standalone:
- Vector embeddings for semantic search
- Optimized for their specific tool
- Not portable across tools
- Can't be shared or committed

## Feature Matrix

| Feature | Repomix | Aider | Stacklit |
|---------|---------|-------|----------|
| Zero config | Yes | Yes | Yes |
| Tree-sitter parsing | Compress mode | Yes | Yes |
| Dependency graph | No | Yes | Yes |
| Committable artifact | No* | No | **Yes** |
| Visual output | No | No | **HTML (static)** |
| Monorepo support | No | No | **Yes** |
| Git activity tracking | No | No | Yes |
| Compact map output | No | No | **Yes (~250 tokens)** |
| Single binary, no deps | No (Node) | No (Python) | Yes (Go) |

*Repomix output is too large to commit meaningfully.

## Real Token Counts

Measured on real open-source projects:

| Repository | Files | Lines | Repomix (full) | Stacklit JSON | Stacklit Compact Map |
|-----------|-------|-------|---------------|---------------|---------------------|
| Express.js | 186 | 14,455 | ~180k tokens | 3,765 tokens | ~200 tokens |
| FastAPI | 392 | 36,714 | ~400k tokens | 5,890 tokens | ~280 tokens |
| Gin | 155 | 19,780 | ~200k tokens | 3,204 tokens | ~220 tokens |
| Axum | 218 | 25,350 | ~280k tokens | 4,120 tokens | ~250 tokens |

Repomix counts estimated from file sizes. Stacklit counts measured directly.

## When to Use What

**Use Repomix if:**
- Your repo is small (<5k lines)
- You're pasting into ChatGPT/Claude web
- You want the simplest possible tool

**Use Stacklit if:**
- You want your AI tools to understand your repo from token zero
- You use multiple AI tools (Claude Code, Cursor, Aider)
- Token efficiency matters (pay-per-token or hitting context limits)
- You want a visual dependency map you can share

## Install

```bash
# Source install from main
curl -fsSL https://raw.githubusercontent.com/liza-mas/stacklit-cli/main/install.sh | sh

# From a local clone
make install
```


*This comparison is maintained by the Stacklit team. We aim to be accurate and fair. If you spot an error or want to add a tool, [open an issue](https://github.com/glincker/stacklit/issues).*
