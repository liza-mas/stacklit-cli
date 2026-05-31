# Code Plan: Parser Map Stage Worker Pool

Task ID: `architecture-main-1-architecture-1-code-planning-0`
Agent ID: `code-planner-2`
Spec reference: `specs/map-reduce.md`
Architecture reference: `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md`

## Source Evidence

- `specs/map-reduce.md`: full goal spec read for global requirement boundaries.
- `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md`: assigned parser map-stage architecture.
- `specs/plans/map-reduce/20260531-072250-architecture-main-1-architecture-0-code-planning-0.md`: prior pre-commit bootstrap plan read for dependency consistency.
- `internal/parser/parser.go`: `ParseAll` currently loops over paths sequentially, appends successful `FileInfo` values in encounter order, collects per-file errors, and continues.
- `internal/parser/treesitter.go`: `TreeSitterParser.Parse` creates gotreesitter parser/tree state inside each parse call.
- `internal/parser/generic.go` and `internal/parser/golang_test.go`: parser tests are package-local and can add focused parse-all coverage without touching engine/config/CLI code.
- SCIP references: `ParseAll` is consumed by `internal/engine/engine.go`, while this scope does not modify that caller.

## Planning Decision

Create one coding task. The assigned scope is cohesive: add the parser-owned worker-count-aware parse-all path, keep `ParseAll(paths)` as one-worker compatibility, and add parser-package tests that prove deterministic ordering, bounded concurrency, non-fatal error collection, and race-detector safety. Splitting implementation and tests would violate TDD colocation; splitting Tree-sitter isolation into a separate task would add dependency overhead without reducing shared-file risk because the same parser API and parser tests need to validate the full behavior.

## Task 1: Add Parser Worker-Pool Parse-All Path

Desc: Add parser-owned worker-pool parsing while preserving one-worker `ParseAll` compatibility.

Done when: Parser package exposes a worker-count-aware parse-all entry point while keeping `ParseAll(paths)` as one-worker compatibility; worker-pool implementation schedules at most the requested positive worker count of concurrent `ParseFile` work, records completions by input index, returns successful `FileInfo` values and per-file errors in input order, keeps jobs/results free of file-content buffers and shared gotreesitter parser/tree/cursor state, and parser tests exercise one-worker compatibility, multi-worker ordering, non-fatal parse-error collection, concurrency bounds via an injected parse function, and `go test -race ./internal/parser` exits 0.

Scope: May edit `internal/parser/parser.go` and add or edit parser-package tests under `internal/parser/*_test.go`; may inspect but should not change `internal/parser/treesitter.go` or `internal/parser/ts_*.go` unless required to eliminate shared gotreesitter parser/tree/cursor state found during implementation. Must not edit `internal/engine/`, `internal/config/`, `internal/cli/`, graph code, docs, benchmarks, or user configuration.

Spec_ref: `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md`

Plan_ref: `specs/plans/map-reduce/20260531-073037-architecture-main-1-architecture-1-code-planning-0.md`

Validation:

- `go test -race ./internal/parser`

Depends on existing task: `architecture-main-1-architecture-0-code-planning-0-coding-0`.

### Implementation Notes

- Add a worker-count-aware parser entry point in `internal/parser/parser.go`, expected shape `ParseAllWithWorkers(paths []string, workerCount int) ([]*FileInfo, []error)`.
- Keep `ParseAll(paths)` as compatibility API by delegating to the worker-count-aware path with one worker.
- Treat user-facing worker-count validation as out of scope. The parser API contract is a positive worker count; direct invalid parser-package calls should not become CLI/config error handling in this task.
- Implement package-private worker-pool machinery that accepts a parse function so tests can inject a blocking parse function and prove the active parse count never exceeds the requested bound. Production should pass `ParseFile`.
- Jobs should carry only input index and path. Results should carry input index plus either `*FileInfo` or `error`. Do not move file bytes, gotreesitter parser, tree, node, or cursor values into shared worker-pool state.
- Build ordered return slices by scanning the indexed completion slots after workers finish, not by appending in completion order.
- Keep parser registry order and `ParseFile` dispatch semantics unchanged.
- Race coverage should exercise real parser-package parallel parsing through the worker-count-aware entry point using temporary source files or existing fixtures.

## Shared-File Audit

| File or module | Task(s) | Dependency handling |
|----------------|---------|---------------------|
| `internal/parser/parser.go` | Task 1 | Single writer in this plan; child task depends on the pre-commit bootstrap coding task. |
| `internal/parser/*_test.go` | Task 1 | Single writer in this plan. |
| `internal/parser/treesitter.go` and `internal/parser/ts_*.go` | Task 1 only if shared gotreesitter state is found | Single writer in this plan; keep unchanged if current per-call isolation is sufficient. |
| `internal/engine/engine.go`, `internal/config/`, `internal/cli/`, graph code, docs, benchmarks, user config | Out of scope for Task 1 | Owned by sibling scopes; no dependency edge from this output task should authorize edits here. |

## Cross-Reference Audit

| Reference | Owner |
|-----------|-------|
| Parser worker-count-aware parse-all API | Task 1 |
| One-worker compatibility for `ParseAll(paths)` | Task 1 |
| Bounded worker scheduling inside `internal/parser/` | Task 1 |
| Input-order successful results and input-order per-file errors | Task 1 |
| Tree-sitter parser/tree/cursor isolation in parser-owned code | Task 1 |
| Parser package race-detector validation | Task 1 |
| Pre-commit gate availability | Existing task `architecture-main-1-architecture-0-code-planning-0-coding-0` |
| `.stacklitrc.json`, CLI flags, engine wiring, graph semantics, docs, and benchmark report | Explicitly out of scope for Task 1 |

## Spec Compliance Matrix

| # | Requirement | Source | Task(s) | Status |
|---|-------------|--------|---------|--------|
| G-1 | Deterministic output for the same repository state and configuration. | `specs/map-reduce.md:33-39` | Task 1 preserves parser result/error order for downstream deterministic reduce behavior; full `generate-json` equivalence is sibling-owned. | Covered |
| G-2 | Parallel parsing must not share mutable Tree-sitter parser or tree instances across workers. | `specs/map-reduce.md:33-39` | Task 1 | Covered |
| G-3 | Memory growth must be bounded by configurable worker count. | `specs/map-reduce.md:33-39` | Task 1 bounds active parser workers and avoids shared file-content buffers; configuration source is sibling-owned. | Covered |
| G-4 | Preserve existing behavior when configured for one worker. | `specs/map-reduce.md:33-39` | Task 1 | Covered |
| G-5 | Feature must be measurable with wall time, CPU profile, and max RSS. | `specs/map-reduce.md:33-39` | Sibling-owned by `architecture-main-1-architecture-3`. | Covered |
| FT1-FR1 | Parse source files through a bounded worker pool when configured parse worker count is greater than one. | `specs/map-reduce.md:76-83` | Task 1 implements parser-side worker pool; config/engine call site is sibling-owned. | Covered |
| FT1-FR2 | Preserve parsed result order as if files were parsed sequentially in input order. | `specs/map-reduce.md:76-83` | Task 1 | Covered |
| FT1-FR3 | Collect per-file parse errors without aborting the full parse, matching current behavior. | `specs/map-reduce.md:76-83` | Task 1 | Covered |
| FT1-FR4 | Do not share `gotreesitter.Parser`, syntax tree, or tree cursor values across workers. | `specs/map-reduce.md:76-83` | Task 1 | Covered |
| FT1-FR5 | Support one-worker mode that exercises the same logical behavior as current sequential implementation. | `specs/map-reduce.md:76-83` | Task 1 | Covered |
| FT1-NFR1 | Parallel parsing must not introduce nondeterministic module ordering, dependency ordering, or JSON output differences. | `specs/map-reduce.md:84-89` | Task 1 preserves parser result order; full JSON-output equivalence is sibling-owned. | Covered |
| FT1-NFR2 | Worker pool implementation must avoid unbounded goroutine creation. | `specs/map-reduce.md:84-89` | Task 1 | Covered |
| FT1-NFR3 | Worker pool implementation must avoid retaining file contents longer than needed for existing parse result contract. | `specs/map-reduce.md:84-89` | Task 1 | Covered |
| FT1-AC1 | One-worker and multi-worker `generate-json` runs produce equivalent normalized index content. | `specs/map-reduce.md:90-96` | Task 1 provides parser-side ordering compatibility; end-to-end `generate-json` coverage is sibling-owned by engine/config integration scope. | Covered |
| FT1-AC2 | Parallel parsing collects parse errors and keeps them non-fatal as in the sequential path. | `specs/map-reduce.md:90-96` | Task 1 | Covered |
| FT1-AC3 | Repeated runs against the same fixture and worker count produce identical normalized outputs. | `specs/map-reduce.md:90-96` | Task 1 provides deterministic parser ordering; end-to-end repeated-run output coverage is sibling-owned. | Covered |
| FT1-AC4 | Race detector reports no data races for parser tests exercising parallel parsing. | `specs/map-reduce.md:90-96` | Task 1 | Covered |
| FT2-FR1 | Provide a way to configure parse worker count for `generate-json`. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-2`; Task 1 only exposes the parser API consumed later. | Covered |
| FT2-FR2 | Treat worker count one as sequential compatibility mode. | `specs/map-reduce.md:127-133` | Task 1 covers parser behavior; config/CLI behavior is sibling-owned by `architecture-main-1-architecture-2`. | Covered |
| FT2-FR3 | Reject invalid worker counts less than one with a clear error. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-2`; parser direct-call validation is not user configuration handling. | Covered |
| FT2-FR4 | Document worker-count control and memory tradeoff. | `specs/map-reduce.md:127-133` | Sibling-owned by `architecture-main-1-architecture-3`. | Covered |
| FT2-NFR1 | Default worker count must be conservative enough to avoid severe memory spikes on large repositories. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-2` and benchmark/docs scope. | Covered |
| FT2-NFR2 | Configuration must be compatible with existing `.stacklitrc.json` files. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-2`. | Covered |
| FT2-NFR3 | Benchmark worker counts 1, 2, 4, and 8 before changing default above one. | `specs/map-reduce.md:134-139` | Sibling-owned by `architecture-main-1-architecture-3`. | Covered |
| FT2-AC1 | `.stacklitrc.json` parse-worker configuration controls parser worker count. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-2`. | Covered |
| FT2-AC2 | Invalid configured worker count exits with clear configuration error. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-2`. | Covered |
| FT2-AC3 | Benchmark results report wall time and max RSS for worker counts 1, 2, 4, and 8. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-3`. | Covered |
| FT2-AC4 | Users can identify how to enable/disable parallel parsing and understand higher worker counts may increase memory use. | `specs/map-reduce.md:140-145` | Sibling-owned by `architecture-main-1-architecture-3`. | Covered |
| ARCH-1 | `ParseAll(paths)` remains the compatibility entry point and represents one-worker logical behavior. | `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md:23-33` | Task 1 | Covered |
| ARCH-2 | Worker-count-aware parser interface accepts an ordered path slice plus a positive worker count. | `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md:23-33` | Task 1 | Covered |
| ARCH-3 | Successful parse results and per-file errors are returned in input order. | `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md:23-33` | Task 1 | Covered |
| ARCH-4 | Worker count bounds goroutine creation and simultaneously active file parses. | `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md:23-33` | Task 1 | Covered |
| ARCH-5 | Parser tests plus race detector validate parser-scope concurrency. | `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md:99-109` | Task 1 | Covered |
| E2E | e2e test coverage for new behavior | Cross-cutting | N/A: Task 1 is parser-package internal and does not wire `generate-json`; end-to-end output equivalence is sibling-owned by engine/config integration scope. | N/A |
| DOC | Documentation updates for changed behavior | Cross-cutting | N/A: Task 1 introduces no user-visible worker-count control; user documentation is sibling-owned by `architecture-main-1-architecture-3`. | N/A |
