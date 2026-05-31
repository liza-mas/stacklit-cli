# Architecture Plan: Parser Map Stage Worker Pool

Status: draft

## Goal

Define the parser-package structure for bounded parallel source-file parsing while preserving one-worker compatibility, deterministic input-order outputs, non-fatal parse-error collection, and Tree-sitter state isolation.

## Context

The parent map-reduce architecture assigns this task to Scope 1, the parser map stage. Today `engine.Run` obtains an ordered file slice from `walker.Walk`, calls `parser.ParseAll(files)`, then performs graph building and index assembly as the sequential reduce phase. `parser.ParseAll` currently loops over paths sequentially, and `ParseFile` reads each file and dispatches to the first parser in the package registry that can parse the path. `TreeSitterParser.Parse` already creates a gotreesitter parser and syntax tree per parse call, so the parser package can own concurrency without moving parse mechanics into engine, CLI, config, graph, or documentation code.

### References

- Goal spec: `specs/map-reduce.md`
- Parent tasks: `architecture-main-1`, `architecture-main-1-architecture-0`
- Parent architecture: `specs/arch-plan/map-reduce/20260531-012947-architecture-main-1.md`
- Bootstrap architecture: `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md`
- Blackboard: `architecture-main-1-architecture-1` task JSON, `architecture-main-1` output summary, active-task summary
- Codebase: `internal/parser/parser.go`, `internal/parser/treesitter.go`, `internal/parser/golang_test.go`, `internal/engine/engine.go`
- Symbol index: SCIP lookup for `ParseAll`, `ParseFile`, and `TreeSitterParser`

### Constraints

- This architecture owns only `internal/parser/` behavior and parser tests.
- Parser code must not resolve `.stacklitrc.json`, parse CLI flags, choose rollout defaults, change engine graph/reduce semantics, or write user documentation.
- `ParseAll(paths)` remains the compatibility entry point and represents one-worker logical behavior.
- The worker-count-aware parser interface accepts an ordered path slice plus a positive worker count; validation of user-provided worker counts stays outside this scope.
- Successful parse results and per-file errors must be returned in input order, matching current downstream expectations.
- No `gotreesitter.Parser`, syntax tree, or tree cursor instance may be stored in shared worker-pool state or shared across workers.
- Worker count bounds goroutine creation and the number of simultaneously active file parses.
- The downstream canonical validation for this scope is `go test -race ./internal/parser`.
- The scope consumes the repo-owned pre-commit quality gate from `architecture-main-1-architecture-0-code-planning-0`.

### Assumptions

- **ASM-PARSER-001**: A parser-package worker-count-aware function can be added without changing current `ParseFile` semantics. *Why*: `ParseFile` already owns file reading, registry dispatch, and parser return values; the worker pool only schedules calls to it. Confidence: HIGH.
- **ASM-PARSER-002**: Registered parser values remain safe to call concurrently as long as they do not share gotreesitter parser, tree, or cursor state. *Why*: `TreeSitterParser` is stateless and creates gotreesitter state per call; Go and generic parsers should be reviewed by the code-planner for mutable receiver state before final implementation. Confidence: MEDIUM.
- **ASM-PARSER-003**: Parser tests can exercise parse-error collection by including an unreadable or otherwise failing path in the input set without changing production error policy. *Why*: current `ParseAll` collects `ParseFile` errors non-fatally and continues. Confidence: HIGH.

### Open Questions

- None for this parser scope. The parent architecture resolves the goal spec's parser-interface question by preserving `ParseAll(paths)` and adding a worker-count-aware parser entry point consumed later by engine/config scope.

---

## Components

### Parser Parse-All API (`internal/parser/parser.go`)

**Responsibility:** Expose parser-package entry points that convert an ordered path slice into parsed `FileInfo` results plus collected per-file errors.

**Boundaries:**
- Exposes: compatibility `ParseAll(paths)` behavior and a worker-count-aware parse-all interface for callers that already resolved a positive worker count.
- Depends on: ordered file paths from callers and `ParseFile` for single-file parsing.

**Key decisions:**
- Keep `ParseAll(paths)` as the one-worker compatibility path. Rationale: existing callers and tests keep their stable API while future engine wiring can opt into configured concurrency through the new interface.
- Put worker-count-aware parsing in `internal/parser/`, not engine. Rationale: parser owns file reads, parser dispatch, parse-error collection, and result ordering; engine should only pass a resolved count in its later scope.
- Treat worker counts less than one as caller-contract violations rather than user-configuration errors. Rationale: config and CLI validation belong to the downstream worker-count control scope, while parser tests should still define deterministic internal behavior for invalid direct calls if the code-planner exposes it.

### Parser Worker Pool (`internal/parser/parser.go`)

**Responsibility:** Bound concurrent calls to `ParseFile` and collect each completed file's result by its input index.

**Boundaries:**
- Exposes: no separate package-level component; it is internal machinery behind the worker-count-aware parse-all interface.
- Depends on: `ParseFile`, a finite path slice, and a positive worker-count limit.

**Key decisions:**
- Represent work and completion records with input indexes. Rationale: workers finish nondeterministically, while the public result slices must preserve the same logical order as sequential parsing.
- Bound goroutine creation by worker count, not by file count. Rationale: the goal spec requires memory growth and active parse work to be bounded by configuration.
- Keep file contents local to each `ParseFile` call and worker execution. Rationale: this avoids retaining file bytes longer than the existing parse-result contract requires.

### Single-File Parser Dispatch (`internal/parser/parser.go`)

**Responsibility:** Read one source file and dispatch its contents through the parser registry.

**Boundaries:**
- Exposes: `ParseFile(path)` as the single-file unit of work used by both compatibility and multi-worker parsing.
- Depends on: `os.ReadFile`, the package parser registry, and parser implementations.

**Key decisions:**
- Do not split file reading out of `ParseFile` for this scope. Rationale: the existing parse contract already scopes file bytes to one path and keeps the worker pool from owning language-specific parsing details.
- Preserve non-fatal error collection at the parse-all boundary. Rationale: `ParseFile` should continue returning a per-file error, and parse-all should decide whether to append it and continue.

### Tree-Sitter Parser Isolation (`internal/parser/treesitter.go` and `internal/parser/ts_*.go`)

**Responsibility:** Keep gotreesitter parser, tree, and cursor lifetimes inside a single file parse.

**Boundaries:**
- Exposes: language-specific `Parser` implementation behavior through the existing parser registry.
- Depends on: gotreesitter grammars and extraction helpers.

**Key decisions:**
- Do not put gotreesitter parser, syntax tree, root node, or tree cursor values in worker-pool shared state. Rationale: the goal spec and external Tree-sitter guidance forbid sharing mutable tree-sitter state concurrently without copying.
- Leave extraction helpers as parser-owned implementation details. Rationale: the worker pool schedules file parses but does not reinterpret AST traversal or language extraction.

### Parser Test Coverage (`internal/parser/*_test.go`)

**Responsibility:** Prove one-worker compatibility, bounded multi-worker behavior, input-order result/error collection, and race-detector safety inside the parser package.

**Boundaries:**
- Exposes: package tests run by `go test -race ./internal/parser`.
- Depends on: existing fixtures or temporary test files created by parser tests.

**Key decisions:**
- Add parser-level tests rather than engine-level output tests in this scope. Rationale: engine/config/CLI equivalence belongs to the dependent worker-count wiring scope.
- Exercise parallel parsing under the race detector. Rationale: absence of shared gotreesitter state and registry concurrency hazards must be validated where concurrency is introduced.

---

## Interfaces

### Engine Reduce Orchestrator -> Parser Parse-All API

**Contract:** A caller provides an ordered `[]string` of file paths and, for the worker-count-aware interface, a positive worker count. The parser package returns successful `[]*FileInfo` and `[]error` in input order as if files were processed sequentially.

**Direction:** Engine calls parser after filesystem walking and before graph building; this scope defines the parser side of that contract, while engine wiring is implemented later.

**Invariants:** Worker count one preserves compatibility behavior; multi-worker parsing changes scheduling only; graph-facing result order, non-fatal error collection, and `FileInfo` shape remain unchanged.

### Parser Worker Pool -> Single-File Parser Dispatch

**Contract:** Each work item carries an input index and path. The worker pool calls `ParseFile(path)` independently for that item and records either one `FileInfo` or one error against the same index.

**Direction:** Worker goroutines call `ParseFile`; collector/reducer code inside `parser.go` rebuilds ordered result and error slices from indexed completions.

**Invariants:** Each input path is attempted at most once per parse-all call; a failing file contributes an error and no `FileInfo`; a successful file contributes a `FileInfo` and no error; collection order is by original input index, not completion order.

### Single-File Parser Dispatch -> Parser Registry

**Contract:** `ParseFile` reads file bytes once, iterates registered parsers in priority order, and invokes the first parser whose `CanParse(path)` returns true.

**Direction:** `ParseFile` calls parser implementations.

**Invariants:** Parser priority and generic fallback behavior remain unchanged; parse-all concurrency does not mutate the registry; per-file parser errors continue to be returned to the parse-all collector.

### Tree-Sitter Parser Isolation -> gotreesitter

**Contract:** Each Tree-sitter-backed file parse creates gotreesitter state for that file and passes only file-local nodes/language/source bytes to extraction helpers during that parse.

**Direction:** `TreeSitterParser.Parse` calls gotreesitter and extraction helpers.

**Invariants:** No gotreesitter parser, syntax tree, node cursor, or cursor-like traversal value escapes into shared worker-pool state or crosses file-parse boundaries.

---

## Data Flow

```text
ordered file paths
  -> parser.ParseAll(paths) compatibility path
       -> worker-count-aware parser path with one worker
  -> parser worker pool when worker count > 1
       worker item: input index + path
       worker: ParseFile(path)
         -> os.ReadFile(path)
         -> registry CanParse/Parse dispatch
         -> TreeSitterParser.Parse creates file-local gotreesitter state when applicable
       completion: input index + FileInfo or error
  -> parser collector restores input-index order
  -> ordered successful []*FileInfo + ordered []error
  -> later engine reduce phase consumes unchanged result shape
```

The concurrent map stage is limited to `internal/parser/`. The reduce phase remains outside this scope and continues to consume the same logical data shape.

---

## Cross-Cutting Concerns

| Concern | Approach |
|---------|----------|
| Error handling | `ParseFile` errors remain per-file and non-fatal at the parse-all boundary; the collector returns errors ordered by input path position. |
| Observability | Parser package does not add logging or metrics in this scope; engine status output continues to report parsed and error counts from returned slices. |
| Configuration | Parser receives only an already-resolved positive worker count through the new interface; CLI/config/default resolution is out of scope. |
| Determinism | Completion order is ignored; indexed collection restores successful results and errors to input order before returning. |
| Memory bounds | Worker count bounds active file reads and parses; worker-pool state stores indexes, paths, `FileInfo` pointers, and errors, not shared file-content buffers. |
| Tree-sitter concurrency | gotreesitter parser/tree/cursor state is created and consumed within a single file parse and never shared in worker-pool state. |
| Testing | Parser tests cover one-worker compatibility, multi-worker bounded parsing, result ordering, non-fatal error collection, and `go test -race ./internal/parser`. |
| Security | No new external command execution or credential-file access is introduced; parser continues to read only file paths supplied by the existing walker/caller pipeline. |

---

## Decomposition

Each scope becomes a code-planning child task.

### Scope 0: Parser Map Stage Worker Pool

**Component(s):** Parser parse-all API, parser worker pool, single-file parser dispatch, Tree-sitter parser isolation, and parser tests.

**Boundary:** Owns worker-pool parsing inside `internal/parser/`; does not resolve user configuration, parse CLI flags, change graph semantics, or write docs.

**Done when:** Parser code supports one-worker compatibility and bounded multi-worker parsing, preserves input-order results and non-fatal parse-error collection, avoids shared gotreesitter parser/tree/cursor state, and parser tests plus race detector exercise parallel parsing without data races.

**Depends on:** Existing task `architecture-main-1-architecture-0-code-planning-0` for the repo-owned pre-commit quality gate.

### Spec Coverage

| Spec Requirement | Scope |
|------------------|-------|
| NFR-000-1 deterministic output for same repository/configuration | Scope 0 preserves parser result order for downstream deterministic reduce behavior; engine-level JSON equivalence is covered by the dependent wiring scope. |
| NFR-000-2 no shared mutable Tree-sitter parser/tree instances | Scope 0 |
| NFR-000-3 memory growth bounded by configurable worker count | Scope 0 bounds active parser workers; configuration source is covered by the dependent wiring scope. |
| NFR-000-4 preserve existing behavior with one worker | Scope 0 |
| FR-001-1 bounded worker pool when configured parse worker count is greater than one | Scope 0 implements the bounded parser-side pool; configuration wiring is covered by the dependent wiring scope. |
| FR-001-2 preserve parsed result order as sequential input order | Scope 0 |
| FR-001-3 collect per-file parse errors without aborting full parse | Scope 0 |
| FR-001-4 do not share gotreesitter parser, syntax tree, or tree cursor across workers | Scope 0 |
| FR-001-5 support one-worker mode with current logical behavior | Scope 0 |
| NFR-001-1 avoid nondeterministic module/dependency/JSON differences | Scope 0 preserves parser ordering for downstream graph/reduce determinism; full output equivalence is covered by the dependent wiring scope. |
| NFR-001-2 avoid unbounded goroutine creation | Scope 0 |
| NFR-001-3 avoid retaining file contents longer than needed | Scope 0 |
| AC-001-2 parse errors collected and non-fatal in parallel path | Scope 0 |
| AC-001-4 race detector reports no parser data races | Scope 0 |
| FR-002-2 worker count one is sequential compatibility mode | Scope 0 for parser behavior; worker-count configuration is covered by the dependent wiring scope. |

### Shared-File Audit

| File or module | Writer scope | Readers |
|----------------|--------------|---------|
| `internal/parser/parser.go` | Scope 0 | Dependent engine/config/CLI wiring scope consumes the parser interface |
| `internal/parser/treesitter.go` and `internal/parser/ts_*.go` | Scope 0 if isolation fixes are needed | Parser tests and dependent runtime scans |
| `internal/parser/*_test.go` | Scope 0 | Parser package validation |
| `internal/engine/engine.go`, `internal/config/`, `internal/cli/`, docs, benchmark artifacts | Out of scope for Scope 0 | Owned by sibling scopes from `architecture-main-1` |

### Dependency Order

Existing task `architecture-main-1-architecture-0-code-planning-0` -> Scope 0.

Scope 0 then feeds the parent architecture's worker-count control and engine-wiring scope by exposing the worker-count-aware parser contract. No sibling scope from this architecture task is emitted.
