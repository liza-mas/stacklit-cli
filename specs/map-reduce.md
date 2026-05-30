# PRD: Map-Reduce Parsing for `generate-json`

Status: draft

## Goal

Reduce `stacklit generate-json` latency on large repositories by parsing source files concurrently while preserving deterministic index output and bounded memory use.

## Context

Profiling `generate-json` against `/home/tangi/Workspace/omni` showed the parse phase dominates runtime. The profiled run took about 2m31s wall time and reached about 4.6GB max RSS. CPU samples showed roughly 91% cumulative time under `parser.(*TreeSitterParser).Parse`, with most time inside `gotreesitter.(*Parser).Parse` and its GLR stack merge/equivalence logic.

The current Stacklit pipeline already has a natural map-reduce shape:

- map: parse each source file into `parser.FileInfo`
- reduce: build the graph, assemble modules, compute metadata, and write JSON

The current implementation performs the map step sequentially.

## General information

Applies to: `stacklit generate-json` and the shared engine/parser path used to produce `stacklit.json`.

### References

- Code: `internal/parser/parser.go` — `ParseAll` loops over file paths sequentially.
- Code: `internal/parser/treesitter.go` — `TreeSitterParser.Parse` creates a new gotreesitter parser per file.
- Code: `internal/engine/engine.go` — `engine.Run` calls `walker.Walk`, then `parser.ParseAll`, then graph/index assembly.
- Profiling artifact: `/tmp/stacklit-generate-json.cpu.pprof` — Omni CPU profile captured from a real generation run.
- Profiling artifact: `/tmp/stacklit-omni-generate-json-flamegraph.png` — flamegraph of the same run.
- External documentation: Tree-sitter parser documentation — syntax trees must not be shared concurrently without copying.

### Non-Functional Requirements

- NFR-000-1: Output must remain deterministic for the same repository state and configuration.
- NFR-000-2: Parallel parsing must not share mutable Tree-sitter parser or tree instances across workers.
- NFR-000-3: Memory growth must be bounded by a configurable worker count.
- NFR-000-4: The implementation must preserve existing behavior when configured for one worker.
- NFR-000-5: The feature must be measurable with wall time, CPU profile, and maximum RSS on a representative large repository.

### Related External Components

- Component C-001 - Tree-sitter/gotreesitter: Parses individual source files.
- Component C-002 - Go runtime scheduler: Executes parse workers concurrently.

### Out of Scope

- Replacing Tree-sitter/gotreesitter.
- Optimizing Tree-sitter internal parse-stack merge/equivalence algorithms.
- Changing module graph semantics.
- Changing the `stacklit.json` schema.
- Adding long-running daemon or cache behavior.
- Introducing per-file incremental parsing.

### Assumptions

- **ASM-000-1**: File-level parsing is independent for Stacklit's current extracted metadata — *Why*: `ParseFile` receives only one path and file contents, and graph assembly happens after all files are parsed — Confidence: HIGH.
- **ASM-000-2**: A bounded worker pool can improve wall time when hot parse work is distributed across many files — *Why*: Omni contains hundreds of parseable files and the current loop is sequential — Confidence: MEDIUM.
- **ASM-000-3**: Worker counts above available CPU or memory capacity may reduce stability or performance — *Why*: the profiled sequential run already had high RSS, and concurrent Tree-sitter parses can multiply live parser state — Confidence: HIGH.

### Open Questions

- **OQ-000-1**: What default worker count should ship? — *Impact if unresolved*: an aggressive default could reduce latency but cause memory spikes on large repositories.
- **OQ-000-2**: Should worker count be exposed as CLI flag, `.stacklitrc.json` key, environment variable, or internal default only? — *Impact if unresolved*: users may not have a stable way to tune performance for machine size.

---

## Feature FT-001 - Parallel File Parsing

### References

- Code: `internal/parser/parser.go` — current sequential `ParseAll`.
- Code: `internal/parser/treesitter.go` — parser-per-file behavior.
- Profiling artifact: `/tmp/stacklit-omni-generate-json-flamegraph.png`.

### Functional Requirements

- FR-001-1: Stacklit must parse source files through a bounded worker pool when the configured parse worker count is greater than one.
- FR-001-2: Stacklit must preserve parsed result order as if files were parsed sequentially in input order.
- FR-001-3: Stacklit must collect per-file parse errors without aborting the full parse, matching current `ParseAll` behavior.
- FR-001-4: Stacklit must not share `gotreesitter.Parser`, syntax tree, or tree cursor values across workers.
- FR-001-5: Stacklit must support a one-worker mode that exercises the same logical behavior as the current sequential implementation.

### Non-Functional Requirements

- NFR-001-1: Parallel parsing must not introduce nondeterministic module ordering, dependency ordering, or JSON output differences for the same input.
- NFR-001-2: Worker pool implementation must avoid unbounded goroutine creation.
- NFR-001-3: Worker pool implementation must avoid retaining file contents longer than needed for the existing parse result contract.

### Acceptance Criteria

- AC-001-1: Given a fixture repository, when `generate-json` runs with one parse worker and with multiple parse workers, then the resulting index content is equivalent except for expected generation metadata such as timestamps.
- AC-001-2: Given a fixture containing at least one file that fails to parse, when parallel parsing runs, then the error is collected and non-fatal as in the sequential path.
- AC-001-3: Given repeated runs against the same fixture and worker count, when outputs are normalized for generation metadata, then the outputs are identical.
- AC-001-4: Given the race detector is run against parser tests, when parallel parsing is exercised, then no data races are reported.

### Depends on

Implementation ordering:

- None.

### Out of Scope

- Per-language worker pools.
- Per-file timeout or cancellation policy.
- Parse result caching.

### Assumptions

- **ASM-001-1**: The existing parser registry can remain process-global if parser instances do not carry mutable per-parse state — *Why*: `TreeSitterParser` is currently stateless and creates a gotreesitter parser inside each parse call — Confidence: MEDIUM.

### Open Questions

- **OQ-001-1**: Should the current `ParseAll` function become parallel by default, or should a new `ParseAllParallel` be introduced and wired from `engine.Run`? — *Impact if unresolved*: affects API churn and test scope.

---

## Feature FT-002 - Worker Count Configuration and Rollout

### References

- Code: `internal/config/config.go` — `.stacklitrc.json` configuration model.
- Code: `internal/cli/generate_json.go` — `generate-json` CLI flags.
- Profiling artifact: `/tmp/stacklit-generate-json.cpu.pprof`.

### Functional Requirements

- FR-002-1: Stacklit must provide a way to configure parse worker count for `generate-json`.
- FR-002-2: Stacklit must treat a worker count of one as sequential compatibility mode.
- FR-002-3: Stacklit must reject invalid worker counts less than one with a clear error.
- FR-002-4: Stacklit must document the worker-count control and its memory tradeoff.

### Non-Functional Requirements

- NFR-002-1: The default worker count must be conservative enough to avoid severe memory spikes on large repositories.
- NFR-002-2: Configuration must be compatible with existing `.stacklitrc.json` files.
- NFR-002-3: Benchmark evidence must compare at least worker counts 1, 2, 4, and 8 on Omni or an equivalent large mixed-language repository before changing the default above one.

### Acceptance Criteria

- AC-002-1: Given `.stacklitrc.json` configures parse workers, when `generate-json` runs, then the configured value controls parser worker count.
- AC-002-2: Given an invalid configured worker count, when `generate-json` runs, then it exits with a clear configuration error.
- AC-002-3: Given worker counts 1, 2, 4, and 8 are benchmarked on the same repository state, then the results report wall time and max RSS for each run.
- AC-002-4: Given documentation is read, then users can identify how to enable/disable parallel parsing and understand that higher worker counts may increase memory use.

### Depends on

Implementation ordering:

- Feature FT-001 - Parallel File Parsing.

### Out of Scope

- Automatic adaptive worker count based on live memory pressure.
- Persistent profiling UI or telemetry.

### Assumptions

- **ASM-002-1**: `.stacklitrc.json` is the likely durable place for worker-count configuration — *Why*: existing performance-related scan settings already live there — Confidence: MEDIUM.

### Open Questions

- **OQ-002-1**: Should CLI flags override `.stacklitrc.json` for worker count? — *Impact if unresolved*: precedence rules may surprise users.
