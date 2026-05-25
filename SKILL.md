---
name: stacklit-navigator
description: Use stacklit.json to navigate codebases without burning tokens on file exploration
---

## When to use

At the start of any session on a repo that has `stacklit.json` at the root. Read it before you read any source files.

## Reading the index

```bash
# Check if index exists
cat stacklit.json | head -5
```

The file has these sections:

- `project` -- repo name, type (single/monorepo), workspaces
- `tech` -- primary language, language breakdown, frameworks
- `modules` -- the map: each module's purpose, files, exports, dependencies, activity level
- `dependencies.edges` -- import relationships between modules
- `dependencies.most_depended` -- the modules everything depends on (touch carefully)
- `hints` -- where to add features, how to run tests, env vars to set

## How to use each section

**Finding code:** Look up the module in `modules` by name. The `purpose` field tells you what it does. The `file_list` tells you exactly which files to read. Skip everything else.

**Understanding structure:** Check `depends_on` and `depended_by` to see how a module connects to the rest. If you need to change something in `src/auth`, check `depended_by` to know what might break.

**Knowing what changed recently:** The `activity` field (high/medium/low) tells you how actively a module is being worked on. `git.hot_files` lists the most-changed files in the last 90 days.

**Adding a feature:** Read `hints.add_feature` first. It tells you which directory to create files in and which files to update.

**Running tests:** Check `hints.test_command`.

## What not to do

- Do not `grep` across all files when `modules` tells you which module to look in
- Do not read entire directories to understand the codebase
- Do not re-explore the same repo every session. The index persists across sessions.
- Do not ignore `most_depended` when planning changes. Those modules have the widest blast radius.

## Using the CLI query commands

Use these commands instead of manually searching the full JSON:

- `stacklit get-module src/auth` -- detailed info on one module
- `stacklit find-module auth` -- search modules by name or purpose
- `stacklit get-dependencies src/auth` -- what it imports and what imports it
- `stacklit get-hot-files` -- most-changed files (90 days)
- `stacklit get-hints` -- conventions, test commands, env vars

## Keeping the index fresh

If you have made significant structural changes (new modules, moved files, changed dependencies):

```bash
stacklit generate-json -o stacklit.json
```

Use `stacklit diff` to check if the index is stale before regenerating.
