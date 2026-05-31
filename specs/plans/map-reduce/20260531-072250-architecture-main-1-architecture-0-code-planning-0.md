# Code Plan: Pre-Commit Bootstrap for Map-Reduce Work

Task ID: `architecture-main-1-architecture-0-code-planning-0`
Agent ID: `code-planner-1`
Spec reference: `specs/map-reduce.md`
Architecture reference: `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md`

## Source Evidence

- `specs/map-reduce.md`: full goal spec read for global requirement boundaries.
- `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md`: assigned support-scope architecture.
- `go.mod`: Go module exists and requires the Go toolchain.
- `Makefile`: repository test target is `go test ./...`.
- Worktree file listing: no existing `.pre-commit-config.yaml`, `requirements-dev.txt`, or equivalent dev-tool manifest was present.
- PyPI index query: latest available `pre-commit` package version in this environment is `4.6.0`.
- Git tag query: latest `pre-commit/pre-commit-hooks` tag returned in this environment is `v6.0.0`.

## Planning Decision

Create one coding task. The scope is support-only and has one observable intent: bootstrap a project-owned pre-commit gate that downstream map-reduce implementation tasks can run. It does not create runtime map-reduce behavior and must not touch parser, engine, CLI, config, docs, or tests.

## Task 1: Bootstrap Project Pre-Commit Gate

Desc: Bootstrap the repository pre-commit gate for map-reduce downstream tasks.

Done when: `.pre-commit-config.yaml` defines local Go hooks for `gofmt` over Go files and `go test ./...`, defines text hygiene hooks for trailing whitespace, end-of-file normalization, YAML syntax, JSON syntax, and markdown-relevant whitespace; `requirements-dev.txt` records `pre-commit==4.6.0`; after provisioning from `requirements-dev.txt` through a project-local environment, `pre-commit run --all-files` exits 0 on the bootstrap commit.

Scope: May create or edit only `.pre-commit-config.yaml` and `requirements-dev.txt`. Must not touch map-reduce parser, engine, CLI, config, docs, tests, generated assets, or unrelated tooling. Must not install OS packages or unrelated global tooling; if `pre-commit` cannot be provisioned from `requirements-dev.txt` in a project-scoped environment, mark BLOCKED with the exact failing command and stderr.

Spec_ref: `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md`

Plan_ref: `specs/plans/map-reduce/20260531-072250-architecture-main-1-architecture-0-code-planning-0.md`

Validation:

- `pre-commit run --all-files`

Depends on: none.

### Implementation Notes

- Prefer `requirements-dev.txt` because no existing project-owned dev-tool manifest was present.
- Record `pre-commit==4.6.0` in `requirements-dev.txt`.
- Define `.pre-commit-config.yaml` with:
  - local hook `gofmt`, `entry: gofmt -w`, `language: system`, `types: [go]`;
  - local hook `go-test`, `entry: go test ./...`, `language: system`, `pass_filenames: false`;
  - `pre-commit/pre-commit-hooks` at `rev: v6.0.0` with `trailing-whitespace`, `end-of-file-fixer`, `check-yaml`, and `check-json`;
  - pass markdown-aware trailing whitespace arguments such as `--markdown-linebreak-ext=md`.
- If pre-commit modifies files on the first validation run, stage the modified in-scope files and run `pre-commit run --all-files` once more.
- Do not add package-manager wrappers, lockfiles, Makefile targets, documentation, or tests unless the task is explicitly revised.

## Shared-File Audit

| File | Task(s) | Dependency handling |
|------|---------|---------------------|
| `.pre-commit-config.yaml` | Task 1 | Single writer; no dependency needed. |
| `requirements-dev.txt` | Task 1 | Single writer; no dependency needed. |

## Cross-Reference Audit

| Reference | Owner |
|-----------|-------|
| Pre-commit configuration | Task 1 |
| Project-scoped pre-commit tooling manifest | Task 1 |
| Parser, engine, CLI, config, docs, and tests | Explicitly out of scope for Task 1 |

## Spec Compliance Matrix

| # | Requirement | Source | Task(s) | Status |
|---|-------------|--------|---------|--------|
| PC-1 | Create repository pre-commit configuration exposing `pre-commit run --all-files`. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:45-56` | Task 1 | Covered |
| PC-2 | Record `pre-commit` as a development dependency in a project-owned manifest. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:58-68` | Task 1 | Covered |
| PC-3 | Include local Go formatting with `gofmt` and local Go tests with `go test ./...`. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:53-55` | Task 1 | Covered |
| PC-4 | Include text hygiene hooks for markdown-relevant trailing whitespace/end-of-file normalization plus YAML and JSON syntax checks. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:28-30` | Task 1 | Covered |
| PC-5 | Do not install OS packages or unrelated global tooling; block if no authorized project-scoped provisioning path exists. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:24-30` | Task 1 | Covered |
| PC-6 | Do not touch parser, engine, CLI, config, docs, or tests. | `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md:22-30` | Task 1 | Covered |
| G-1 | Deterministic output for same repository state and configuration. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-1` / `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| G-2 | Parallel parsing must not share mutable Tree-sitter parser or tree instances across workers. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| G-3 | Memory growth must be bounded by configurable worker count. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-1` / `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| G-4 | One-worker mode preserves existing behavior. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-1` / `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| G-5 | Feature must be measurable with wall time, CPU profile, and max RSS. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| FT1-FR1 | Parse source files through a bounded worker pool when worker count is greater than one. | `specs/map-reduce.md:76-83` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-FR2 | Preserve parsed result order as if files were parsed sequentially in input order. | `specs/map-reduce.md:76-83` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-FR3 | Collect per-file parse errors without aborting full parse, matching current behavior. | `specs/map-reduce.md:76-83` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-FR4 | Do not share `gotreesitter.Parser`, syntax tree, or tree cursor values across workers. | `specs/map-reduce.md:76-83` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-FR5 | Support one-worker mode that exercises current sequential logical behavior. | `specs/map-reduce.md:76-83` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-NFR1 | Do not introduce nondeterministic module ordering, dependency ordering, or JSON output differences. | `specs/map-reduce.md:84-89` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-NFR2 | Avoid unbounded goroutine creation. | `specs/map-reduce.md:84-89` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-NFR3 | Avoid retaining file contents longer than needed for the existing parse result contract. | `specs/map-reduce.md:84-89` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-AC1 | One-worker and multi-worker `generate-json` runs produce equivalent normalized index content. | `specs/map-reduce.md:90-96` | Sibling-owned by `architecture-main-1-architecture-1` / integration tests; outside Task 1 scope. | Covered by sibling scope |
| FT1-AC2 | Parallel parsing collects non-fatal parse errors as the sequential path does. | `specs/map-reduce.md:90-96` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT1-AC3 | Repeated runs against same fixture and worker count produce identical normalized outputs. | `specs/map-reduce.md:90-96` | Sibling-owned by `architecture-main-1-architecture-1` / integration tests; outside Task 1 scope. | Covered by sibling scope |
| FT1-AC4 | Race detector reports no data races for parser tests exercising parallel parsing. | `specs/map-reduce.md:90-96` | Sibling-owned by `architecture-main-1-architecture-1`; outside Task 1 scope. | Covered by sibling scope |
| FT2-FR1 | Provide a way to configure parse worker count for `generate-json`. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-FR2 | Treat worker count of one as sequential compatibility mode. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-FR3 | Reject invalid worker counts less than one with a clear error. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-FR4 | Document worker-count control and memory tradeoff. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| FT2-NFR1 | Default worker count must avoid severe memory spikes on large repositories. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-2` / `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| FT2-NFR2 | Configuration must be compatible with existing `.stacklitrc.json` files. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-NFR3 | Benchmark worker counts 1, 2, 4, and 8 before changing default above one. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| FT2-AC1 | `.stacklitrc.json` parse-workers configuration controls parser worker count. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-AC2 | Invalid configured worker count exits with clear configuration error. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-2`; outside Task 1 scope. | Covered by sibling scope |
| FT2-AC3 | Benchmark results report wall time and max RSS for worker counts 1, 2, 4, and 8. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| FT2-AC4 | Users can identify how to enable/disable parallel parsing and understand memory use. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-3`; outside Task 1 scope. | Covered by sibling scope |
| E2E | e2e test coverage for new behavior | Cross-cutting | N/A: Task 1 is a support-only quality-gate bootstrap and introduces no runtime product behavior. | N/A |
| DOC | Documentation updates for changed behavior | Cross-cutting | N/A: Task 1 introduces no user-visible `generate-json` behavior; runtime docs are sibling-owned by `architecture-main-1-architecture-3`. | N/A |
