# Architecture Plan: Map-Reduce Parsing for generate-json

Status: draft

## Goal

Define the shared structure for bounded parallel parsing in `stacklit generate-json` while preserving deterministic output, one-worker compatibility, explicit worker-count rollout controls, and benchmark evidence that includes wall time, CPU profile, and maximum RSS.

## Context

`engine.Run` currently walks the repository, calls `parser.ParseAll(files)`, then builds graph, metadata, and JSON outputs. `parser.ParseAll` reads and parses file paths sequentially. Tree-sitter parsing creates a new gotreesitter parser inside each file parse, which gives the existing parser package a natural file-level map boundary. Graph construction and schema assembly already behave as the reduce phase and remain sequential.

### References

- Goal spec: `specs/map-reduce.md`
- Parent tasks: none
- Codebase: `internal/parser/parser.go`, `internal/parser/treesitter.go`, `internal/engine/engine.go`, `internal/config/config.go`, `internal/cli/generate_json.go`, `internal/graph/graph.go`, `internal/renderer/stable.go`, `internal/config/config_test.go`, `internal/engine/engine_test.go`, `internal/cli/index_query_test.go`, `README.md`, `USAGE.md`, `Makefile`, `go.mod`
- Prior review feedback: architecture rejection for missing CPU profile evidence in rollout scope and output contract.

### Constraints

- The `stacklit.json` schema and graph semantics do not change.
- Parsed results must appear to downstream graph assembly in the same relative order as the input file list.
- Per-file parse errors remain non-fatal and collected.
- No mutable `gotreesitter.Parser`, syntax tree, or tree cursor instance crosses worker boundaries.
- Worker count must be positive; worker count one is the compatibility mode.
- The default worker count remains `1` until benchmark evidence justifies raising it.
- Configuration must be backward-compatible with existing `.stacklitrc.json` files.
- `generate-json --multi` must not introduce a separate worker-count behavior; each repo scan uses the same resolved worker-count contract as single-repo scans.
- Rollout benchmark evidence must include wall time, CPU profile, and maximum RSS for the compared worker counts.

### Assumptions

- **ASM-ARCH-001**: The existing walker returns the full file list before parsing and can continue to be the sole producer of parse work items. *Why*: `engine.Run` already materializes `files` before calling `parser.ParseAll`. Confidence: HIGH.
- **ASM-ARCH-002**: The parser registry can remain process-global because registered parser values do not hold per-parse mutable state that is shared by gotreesitter. *Why*: `TreeSitterParser.Parse` creates the gotreesitter parser and tree per call. Confidence: MEDIUM.
- **ASM-ARCH-003**: Keeping graph assembly sequential is enough to preserve module and dependency determinism. *Why*: `graph.Build` and downstream assembly already sort exposed slices and maps where output order matters. Confidence: HIGH.
- **ASM-ARCH-004**: CPU profile evidence can be captured as benchmark evidence without changing runtime index schema or graph behavior. *Why*: the goal spec cites existing CPU profile artifacts as measurement evidence rather than as a product surface. Confidence: HIGH.

### Open Questions

- None. The goal spec open questions are resolved structurally here: default worker count is `1`, control is exposed through `.stacklitrc.json` plus a `generate-json` CLI override, CLI override takes precedence over config, and the existing `ParseAll` remains as a compatibility entry point backed by a new worker-count-aware parser interface.

---

## Components

### Parser Map Stage (`internal/parser/`)

**Responsibility:** Own file-level parsing and the bounded worker pool that maps input paths to `parser.FileInfo` results.

**Boundaries:**
- Exposes: compatibility `ParseAll(paths)` behavior and a worker-count-aware parse-all interface for callers that resolve configuration.
- Depends on: file paths from the walker and per-file parser implementations.

**Key decisions:**
- Preserve the existing `ParseAll(paths)` contract as one-worker compatibility. Rationale: current tests and callers keep a stable API while `engine.Run` can opt into configured concurrency.
- Add a worker-count-aware parser entry point owned by the parser package. Rationale: the parser package owns file reads, parser dispatch, error collection, and result ordering, so concurrency belongs inside this boundary rather than in engine orchestration.
- Represent work results with their input index internally. Rationale: workers can finish nondeterministically, but the reducer can emit successful results and errors in input order before graph assembly.
- Do not share gotreesitter parser, tree, or cursor instances across workers. Rationale: the goal spec requires per-worker parse isolation and Tree-sitter syntax tree values are not a shared concurrency boundary.

### Engine Reduce Orchestrator (`internal/engine/`)

**Responsibility:** Own the map-reduce pipeline boundary: walk files, resolve parse worker count, invoke parser map stage, then perform graph/index assembly as the reduce stage.

**Boundaries:**
- Exposes: `Run` and `RunMulti` behavior for CLI commands and tests.
- Depends on: `config` for resolved worker-count configuration, `walker` for input files, `parser` for parsed file metadata, and existing graph/schema/rendering packages for reduce output.

**Key decisions:**
- Keep graph building, metadata assembly, Merkle computation, insights application, and rendering sequential. Rationale: the spec targets parse latency only and explicitly excludes graph semantic changes.
- Resolve a single positive parse worker count before calling the parser package. Rationale: parser code should not know about CLI precedence or `.stacklitrc.json` semantics.
- Use the same worker-count path for `RunMulti` and single-repo `Run`. Rationale: `generate-json --multi` is still a `generate-json` scan and should not fork rollout behavior.

### Worker Count Configuration (`internal/config/` and `internal/cli/generate_json.go`)

**Responsibility:** Own the user-facing worker-count control, validation, and precedence.

**Boundaries:**
- Exposes: `.stacklitrc.json` schema support for a positive `parse_workers` value and a `generate-json --parse-workers` override.
- Depends on: Cobra flag parsing in CLI and config loading in engine.

**Key decisions:**
- Add `parse_workers` to `.stacklitrc.json` as the durable configuration key. Rationale: existing scan and output controls already live in config.
- Add a `--parse-workers` flag to `generate-json` with CLI override precedence. Rationale: benchmarking and one-off large-repo tuning need a command-local control without editing config.
- Treat omitted worker count as `1` and reject configured or flagged values less than one with a clear error. Rationale: one worker preserves current behavior and invalid values should not silently fall back.
- Document that higher values may increase maximum RSS. Rationale: memory tradeoff is part of the feature contract.

### Rollout Evidence and User Documentation (`README.md`, `USAGE.md`, benchmark artifact)

**Responsibility:** Own user-facing worker-count documentation and benchmark evidence for the rollout decision.

**Boundaries:**
- Exposes: command reference, configuration reference, memory tradeoff notes, and a benchmark report comparing worker counts.
- Depends on: implemented parser and worker-count controls.

**Key decisions:**
- Keep default worker count at `1` in this goal. Rationale: NFR-002-3 requires benchmark evidence before changing the default above one, so the implementation can ship safely before later default tuning.
- Store benchmark evidence as a repo-tracked markdown artifact under `specs/benchmarks/map-reduce/` with references to any retained CPU profile artifacts. Rationale: benchmark results are acceptance evidence, not runtime behavior, and need reviewable provenance.
- Require each benchmarked worker count to report wall time, maximum RSS, and CPU profile evidence from the same repository state. Rationale: NFR-000-5 is broader than AC-002-3, and downstream planning must not be able to satisfy rollout evidence while omitting CPU profiles.

### Pre-Commit Bootstrap (`.pre-commit-config.yaml` and project-scoped tooling manifest)

**Responsibility:** Own repository quality-gate bootstrap only if the repository still lacks a pre-commit configuration when downstream planning runs.

**Boundaries:**
- Exposes: a runnable pre-commit gate for subsequent code tasks.
- Depends on: an authorized project-scoped tooling path.

**Key decisions:**
- This scope is dependency-only support for all implementation scopes and must not change map-reduce runtime behavior. Rationale: it exists to satisfy the pipeline's bootstrap-precommit contract while keeping feature scopes clean.

---

## Interfaces

### Engine Reduce Orchestrator -> Parser Map Stage

**Contract:** Engine passes the ordered file path slice and a resolved positive worker count to the parser package. Parser returns successful `[]*parser.FileInfo` and `[]error` as if the input had been processed sequentially.

**Direction:** Engine calls parser after filesystem walking and before graph building.

**Invariants:** Result order is by input path order for successfully parsed files; errors are collected without aborting; worker count one is compatibility mode; no parser-owned mutable Tree-sitter state is shared across worker boundaries.

### CLI -> Engine Reduce Orchestrator

**Contract:** `generate-json` converts `--parse-workers` into an optional override and passes it to `engine.Run` or `engine.RunMulti`; absence of the flag leaves config/default resolution to engine and config.

**Direction:** CLI calls engine.

**Invariants:** CLI override takes precedence over `.stacklitrc.json`; values less than one fail before parsing; `--multi` uses the same override contract for each repo scan.

### Config -> Engine Reduce Orchestrator

**Contract:** Config exposes a resolved parse worker count with default `1` and validates explicit values.

**Direction:** Engine asks config for scan settings, then passes only the resolved count to parser.

**Invariants:** Existing config files without `parse_workers` remain valid; existing malformed-config fallback behavior is not broadened into silent acceptance of invalid explicit worker counts.

### Parser Map Stage -> Tree-Sitter/gotreesitter

**Contract:** Each file parse creates and uses its own gotreesitter parser/tree/cursor lifetime inside the worker processing that file.

**Direction:** Parser package calls gotreesitter through `TreeSitterParser.Parse`.

**Invariants:** No gotreesitter parser, syntax tree, or tree cursor instance is stored in shared worker-pool state or returned to callers.

### Rollout Evidence -> Worker Count Configuration

**Contract:** Benchmark evidence records wall time, CPU profile evidence, and max RSS for worker counts 1, 2, 4, and 8 on the same large repository state before any future default above one is proposed.

**Direction:** Documentation and benchmark report consume the implemented controls.

**Invariants:** Benchmarking does not change runtime schema, graph semantics, or default worker count in this goal. CPU profile evidence is an artifact of benchmark execution and not a new user-facing telemetry feature.

---

## Data Flow

```text
generate-json CLI
  -> parse --parse-workers override
  -> engine.Run / engine.RunMulti
  -> config.Load and worker-count resolution
  -> walker.Walk produces ordered file paths
  -> parser map stage runs bounded workers
       each worker: ParseFile -> GoParser / TreeSitterParser / GenericParser
       collector: restore input-order result and error slices
  -> graph.Build and assembleIndex reduce parsed metadata
  -> renderer.WriteJSON preserves stable semantic output
```

The only concurrent portion is the parser map stage. The reduce phase receives the same logical data shape it receives today.

---

## Cross-Cutting Concerns

| Concern | Approach |
|---------|----------|
| Error handling | Parser keeps per-file errors non-fatal and ordered; config/CLI validation errors fail clearly before parse work starts. |
| Observability | Existing quiet/non-quiet status output remains unchanged except parsed count reflects parser results; benchmark evidence records wall time, CPU profile evidence, and max RSS externally. |
| Configuration | `.stacklitrc.json` owns durable `parse_workers`; `generate-json --parse-workers` owns command override; default remains `1`. |
| Determinism | Parser collector restores input-order slices; graph and renderer retain existing stable ordering behavior. |
| Memory bounds | Worker count is the concurrency bound; parser workers read and parse one file at a time and do not retain file contents beyond `FileInfo` extraction. |
| Testing | Parser tests cover one-worker and multi-worker behavior, parse-error collection, deterministic ordering, and race detector; engine/CLI/config tests cover resolution, override precedence, invalid values, and output equivalence; docs/benchmark scope covers rollout evidence including CPU profiles. |
| Security | No new external input execution paths; user-provided worker count is validated as a positive integer; no secrets or credential files are read. |

---

## Systemic Decomposition Review

No systemic issues identified.

The prior rollout-evidence blind spot is closed in this draft by making CPU profile evidence explicit in the rollout component boundary, interface contract, cross-cutting observability/testing concerns, Scope 3 done_when, output metadata, and validation/artifact expectations. The main load-bearing decision remains the parser/engine interface: downstream scopes must treat it as the single contract for worker-count-aware parsing rather than redefining concurrency at the CLI or engine layer.

---

## Decomposition

Each scope becomes a code-planning child task. Scope 0 is emitted only because this worktree lacks `.pre-commit-config.yaml` and no active repo-wide `bootstrap-precommit` task exists outside this architecture task.

### Scope 0: Pre-Commit Bootstrap

**Component(s):** Pre-commit bootstrap support.

**Boundary:** Owns `.pre-commit-config.yaml` and any project-scoped tooling manifest required to run pre-commit. This task may provision pre-commit only via a project-scoped mechanism already available in the repo or explicitly defined by the task/config. Do not install OS packages or unrelated global tooling. If no authorized project-scoped path exists, mark BLOCKED. Hook categories should cover Go formatting/tests through repo-owned commands plus general text hygiene for markdown, YAML, JSON, and trailing whitespace. Does not touch map-reduce parser, engine, CLI, config, docs, or tests.

**Done when:** A project-scoped pre-commit configuration exists, pre-commit is recorded as a dev dependency in the selected project-owned tooling manifest or lockfile, and the repo-owned pre-commit runner exits 0 against the bootstrap commit.

**Depends on:** none.

### Scope 1: Parser Map Stage

**Component(s):** Parser package and parser tests.

**Boundary:** Owns worker-pool parsing inside `internal/parser/`; does not resolve user configuration, parse CLI flags, change graph semantics, or write docs.

**Done when:** Parser code supports one-worker compatibility and bounded multi-worker parsing, preserves input-order results and non-fatal parse-error collection, avoids shared gotreesitter parser/tree/cursor state, and parser tests plus race detector exercise parallel parsing without data races.

**Depends on:** Scope 0.

### Scope 2: Worker Count Control and Engine Wiring

**Component(s):** Config package, `generate-json` CLI, engine orchestration, and integration tests.

**Boundary:** Owns `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, CLI-over-config precedence, engine-to-parser worker-count wiring for single and multi repo scans, and output equivalence tests. Does not implement parser worker-pool internals or write user documentation/benchmark reports.

**Done when:** `generate-json` resolves worker count from default/config/CLI with invalid values rejected, passes the resolved count into the parser map stage for both single and multi scans, and integration tests prove worker count one and multiple workers produce equivalent normalized indexes for fixtures.

**Depends on:** Scope 1.

### Scope 3: Rollout Documentation and Benchmark Evidence

**Component(s):** User docs and benchmark artifact.

**Boundary:** Owns README/USAGE updates and a repo-tracked benchmark report with wall time, CPU profile evidence, and max RSS for worker counts 1, 2, 4, and 8. Does not change parser, config, CLI, engine, graph, or renderer behavior.

**Done when:** Documentation explains `.stacklitrc.json` and CLI worker controls plus memory tradeoffs, and benchmark evidence reports wall time, CPU profile evidence, and max RSS for worker counts 1, 2, 4, and 8 on Omni or an equivalent large mixed-language repository at a fixed repository state.

**Depends on:** Scope 2.

### Spec Coverage

| Spec Requirement | Scope |
|------------------|-------|
| NFR-000-1 deterministic output | Scope 1, Scope 2 |
| NFR-000-2 no shared mutable Tree-sitter parser/tree instances | Scope 1 |
| NFR-000-3 memory bounded by worker count | Scope 1, Scope 2 |
| NFR-000-4 one-worker preserves existing behavior | Scope 1, Scope 2 |
| NFR-000-5 measurable with wall time, CPU profile, max RSS | Scope 3 |
| FR-001-1 bounded worker pool when workers > 1 | Scope 1 |
| FR-001-2 preserve parsed result order | Scope 1 |
| FR-001-3 collect per-file parse errors non-fatally | Scope 1 |
| FR-001-4 no shared gotreesitter parser/tree/cursor across workers | Scope 1 |
| FR-001-5 one-worker mode exercises current logical behavior | Scope 1 |
| NFR-001-1 no nondeterministic module/dependency/JSON differences | Scope 1, Scope 2 |
| NFR-001-2 avoid unbounded goroutine creation | Scope 1 |
| NFR-001-3 avoid retaining file contents longer than needed | Scope 1 |
| AC-001-1 one worker and multiple workers produce equivalent index content | Scope 2 |
| AC-001-2 parse errors collected and non-fatal in parallel path | Scope 1 |
| AC-001-3 repeated runs normalized for metadata are identical | Scope 2 |
| AC-001-4 race detector reports no parser races | Scope 1 |
| FR-002-1 provide worker-count configuration | Scope 2 |
| FR-002-2 worker count one is sequential compatibility mode | Scope 1, Scope 2 |
| FR-002-3 reject invalid worker counts less than one | Scope 2 |
| FR-002-4 document worker-count control and memory tradeoff | Scope 3 |
| NFR-002-1 conservative default avoids severe memory spikes | Scope 2 |
| NFR-002-2 compatible with existing `.stacklitrc.json` files | Scope 2 |
| NFR-002-3 benchmark 1, 2, 4, 8 before changing default above one | Scope 3 |
| AC-002-1 configured parse workers control parser worker count | Scope 2 |
| AC-002-2 invalid configured count exits with clear config error | Scope 2 |
| AC-002-3 benchmark report includes wall time and max RSS | Scope 3 |
| AC-002-4 users can identify enable/disable control and memory risk | Scope 3 |

### Shared-File Audit

| File or module | Writer scope | Readers |
|----------------|--------------|---------|
| `.pre-commit-config.yaml` and project-owned pre-commit tooling manifest | Scope 0 | Scopes 1, 2, 3 as quality gate users |
| `internal/parser/` implementation and parser tests | Scope 1 | Scope 2 consumes parser interface |
| `internal/config/config.go` and `internal/config/config_test.go` | Scope 2 | Scope 3 reads documented behavior |
| `internal/cli/generate_json.go` and CLI tests | Scope 2 | Scope 3 reads documented behavior |
| `internal/engine/engine.go` and engine tests | Scope 2 | Scope 3 reads benchmarkable behavior |
| `README.md`, `USAGE.md`, `specs/benchmarks/map-reduce/` | Scope 3 | none |

### Dependency Order

Scope 0 -> Scope 1 -> Scope 2 -> Scope 3.

Scope 0 is support-only. Scope 1 owns the parser interface consumed by Scope 2. Scope 2 owns the runtime controls that Scope 3 documents and benchmarks.
