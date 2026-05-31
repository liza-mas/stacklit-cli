# Code Plan: Worker Count Control and Engine Wiring

Task ID: `architecture-main-1-architecture-2-code-planning-0`
Agent ID: `code-planner-1`
Spec reference: `specs/map-reduce.md`
Architecture reference: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`

## Source Evidence

- `specs/map-reduce.md`: full goal spec read for global requirement boundaries.
- `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`: assigned worker-count control architecture.
- `specs/plans/map-reduce/20260531-073037-architecture-main-1-architecture-1-code-planning-0.md`: upstream parser code plan read for parser API dependency and scope consistency.
- `internal/config/config.go`: `Config` currently has top-level scan settings, `DefaultConfig`, and `Load(root) *Config` that merges `.stacklitrc.json` onto defaults while preserving compatibility for omitted keys.
- `internal/config/config_test.go`: config tests currently cover defaults, custom values, malformed config fallback, and generated-output ignore patterns.
- `internal/cli/generate_json.go`: `generate-json` currently forwards `output`, `workspace`, `multi`, and `insights` options into `engine.Run` or `engine.RunMulti`.
- `internal/cli/index_query_test.go`: CLI tests already instantiate `newGenerateJSONCmd` and execute fixture `generate-json` commands.
- `internal/engine/engine.go`: `Run` loads config, walks files, calls `parser.ParseAll(files)`, then performs reduce/output assembly; `RunMulti` reads repo paths and calls `Run` for each repo.
- `internal/engine/engine_test.go`: engine tests already cover single-repo JSON output behavior, config-driven output paths, workspace roots, insights, and multi-index output formatting.
- `internal/parser/parser.go`: current checkout exposes only `ParseAll(paths)`; this plan depends on upstream parser coding task `architecture-main-1-architecture-1-code-planning-0-coding-0` to provide the worker-count-aware parse-all entry point planned by the parser scope.

## Planning Decision

Create four coding tasks. The assigned scope crosses config, engine, CLI, and integration test boundaries, so splitting by ownership keeps each child task single-intent while preserving dependency order where shared files or runtime contracts require it. Config support is first because engine resolution needs a validated config value. Engine wiring is second because CLI forwarding needs engine option fields and because output equivalence tests need the resolved worker-count path. CLI forwarding and engine-level equivalence coverage can run after engine wiring; they touch different source files except that equivalence coverage extends `internal/engine/engine_test.go`, so it depends on the engine wiring task.

## Task 1: Add Parse Worker Configuration

Desc: Add `.stacklitrc.json` parse worker configuration defaults and validation.

Done when: Config loading exposes a top-level JSON key `parse_workers`, `DefaultConfig` yields parse worker count `1`, existing `.stacklitrc.json` files without `parse_workers` still load with default `1`, explicit configured values less than one are rejected through a validation-capable config path with a clear error that names `parse_workers`, and config tests cover default, omitted, valid configured, explicit zero, explicit negative, and compatibility with existing custom config.

Scope: May edit `internal/config/config.go` and `internal/config/config_test.go`. Must not edit CLI, engine, parser internals, user documentation, benchmark artifacts, or output schemas. Preserve existing `Load(root) *Config` compatibility unless changing it is required by local compile-time API constraints; any new validation-capable API must be consumed by later engine work rather than by config knowing about CLI precedence.

Spec_ref: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`

Plan_ref: `specs/plans/map-reduce/20260531-074008-architecture-main-1-architecture-2-code-planning-0.md`

Validation:

- `go test ./internal/config`

Depends on existing tasks: `architecture-main-1-architecture-0-code-planning-0-coding-0-gofmt-prereq`, `architecture-main-1-architecture-0-code-planning-0-coding-0-precommit-bootstrap`.

### Implementation Notes

- Treat omission of `parse_workers` as default `1`; treat explicit JSON values `0` or negative as invalid because the spec and architecture require configured worker counts less than one to be rejected.
- If preserving `Load(root) *Config`, add a validation-capable loader or helper for engine use, for example `LoadValidated(root) (*Config, error)` or an equivalent API that can distinguish explicit invalid `parse_workers` from omission.
- Keep `parse_workers` top-level alongside `max_depth`, `max_modules`, and `max_exports`.
- Keep malformed unrelated config behavior compatible with existing tests unless the implementation has a narrower, documented reason to surface parse-worker validation errors.

## Task 2: Wire Engine Parse Worker Resolution

Desc: Wire resolved parse worker counts through engine single- and multi-repo scans.

Done when: `engine.Options` and `engine.MultiOptions` can carry an optional CLI-sourced parse worker override, `Run` resolves exactly one positive worker count using CLI override first, then `.stacklitrc.json` `parse_workers`, then default `1`, invalid config or override values return clear errors before parser work starts, `Run` calls the upstream parser worker-count-aware parse-all entry point with the resolved count, `RunMulti` forwards the same override into each `Run` call while absent override lets each repo use its own config/default, worker-count config failures in multi-repo scans are fatal rather than warning-only skips, and engine tests cover default resolution, config resolution, CLI-over-config precedence, invalid config, invalid override, and multi-repo forwarding/error behavior.

Scope: May edit `internal/engine/engine.go` and `internal/engine/engine_test.go`; may consume the config API from Task 1 and the parser worker-count-aware parse-all API from `architecture-main-1-architecture-1-code-planning-0-coding-0`. Must not edit `internal/parser/` worker-pool internals, CLI command construction, user documentation, benchmark artifacts, graph/reduce semantics, renderer/schema formats, or `.stacklitrc.json` documentation.

Spec_ref: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`

Plan_ref: `specs/plans/map-reduce/20260531-074008-architecture-main-1-architecture-2-code-planning-0.md`

Validation:

- `go test ./internal/config ./internal/engine`

Depends on sibling tasks: Task 1.

Depends on existing tasks: `architecture-main-1-architecture-1-code-planning-0-coding-0`.

### Implementation Notes

- Use an internal absence sentinel that cannot be a valid user value, such as `0`, only for option fields; supplied `0` remains invalid when it comes from CLI or explicit config.
- Resolve worker count after loading config and before `walker.Walk` and parser invocation.
- Keep reduce behavior unchanged: graph building, metadata assembly, insights, Merkle computation, and rendering should not branch on worker count.
- Preserve normal non-worker-count multi-repo scan behavior unless the error is a worker-count configuration/override error covered by this scope.
- Engine tests may use package-local helpers or temporary fixture repos, but they should exercise `Run` and `RunMulti` package APIs instead of parser internals.

## Task 3: Expose Generate-JSON Parse Worker Override

Desc: Expose and forward `generate-json --parse-workers` as a CLI override.

Done when: `generate-json` registers an integer `--parse-workers` flag, command execution uses Cobra flag presence to distinguish absent override from a supplied invalid value, supplied positive values are forwarded to `engine.Run` and `engine.RunMulti`, supplied values less than one return a clear CLI error that names `--parse-workers`, absent flag behavior preserves config/default resolution, and CLI tests cover flag registration, invalid supplied zero/negative values, single-repo CLI-over-config precedence, and `--multi` forwarding of the same override.

Scope: May edit `internal/cli/generate_json.go` and `internal/cli/index_query_test.go`. Must not edit config loading internals, engine resolution internals beyond consuming Task 2 option fields, parser internals, user documentation, benchmark artifacts, or output schemas.

Spec_ref: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`

Plan_ref: `specs/plans/map-reduce/20260531-074008-architecture-main-1-architecture-2-code-planning-0.md`

Validation:

- `go test ./internal/cli ./internal/engine`

Depends on sibling tasks: Task 2.

### Implementation Notes

- Register the flag on `newGenerateJSONCmd` near the existing `output`, `workspace`, `multi`, and `insights` flags.
- Use `cmd.Flags().Changed("parse-workers")` when deciding whether to populate the engine override field.
- CLI tests should prefer public command execution and observable generated output/errors over reaching into engine internals.
- For CLI-over-config tests, create fixture `.stacklitrc.json` files that configure a different worker count than the supplied CLI flag.

## Task 4: Add Normalized Output Equivalence Coverage

Desc: Add engine-level normalized output equivalence tests for one-worker and multi-worker scans.

Done when: Engine integration tests run fixture repositories through worker count `1` and a worker count greater than `1` for single-repo `Run` and multi-repo `RunMulti` paths, normalize only expected generation metadata such as timestamps, assert semantic `schema.Index` content and `schema.MultiIndex` content are equivalent, assert repeated runs with the same worker count produce identical normalized outputs, and `go test ./internal/engine` exercises those comparisons.

Scope: May edit `internal/engine/engine_test.go` and add engine-package test helpers in `internal/engine/*_test.go` if needed. Must not edit production graph/reduce logic, parser internals, config/CLI production code, user documentation, benchmark artifacts, or output schemas.

Spec_ref: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`

Plan_ref: `specs/plans/map-reduce/20260531-074008-architecture-main-1-architecture-2-code-planning-0.md`

Validation:

- `go test ./internal/engine`

Depends on sibling tasks: Task 2.

### Implementation Notes

- Compare `Result.Index` values directly for single-repo coverage after blanking or normalizing generation metadata fields that are expected to vary.
- For multi-repo coverage, read the generated multi-index JSON from `RunMulti` output and normalize generation metadata before comparison.
- Do not normalize module, dependency, structure, language, repo summary, framework, or hint content; those are semantic outputs that must remain equivalent.
- Keep fixtures small but mixed enough to exercise parser scheduling through more than one file.

## Shared-File Audit

| File or module | Task(s) | Dependency handling |
|----------------|---------|---------------------|
| `internal/config/config.go` | Task 1 | Single writer in this plan. |
| `internal/config/config_test.go` | Task 1 | Single writer in this plan. |
| `internal/engine/engine.go` | Task 2 | Single writer in this plan; Task 2 depends on Task 1 and upstream parser coding task. |
| `internal/engine/engine_test.go` | Task 2, Task 4 | Task 4 depends on Task 2 because both edit this test file. |
| `internal/cli/generate_json.go` | Task 3 | Single writer in this plan; depends on Task 2 for engine option fields. |
| `internal/cli/index_query_test.go` | Task 3 | Single writer in this plan. |
| `internal/parser/` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Task 2 consumes the worker-count-aware parser API only and must not modify parser internals. |
| `README.md`, `USAGE.md`, `specs/benchmarks/map-reduce/` | Out of scope for this plan | Owned by downstream rollout documentation/benchmark scope `architecture-main-1-architecture-3`. |
| `.pre-commit-config.yaml` and project-scoped pre-commit tooling | Existing bootstrap tasks | Task 1 waits for bootstrap/gofmt tasks; later sibling tasks depend transitively through Task 1 and parser coding where relevant. |

## Cross-Reference Audit

| Reference | Owner |
|-----------|-------|
| `.stacklitrc.json` `parse_workers` schema/default/validation | Task 1 |
| Config validation-capable API for invalid explicit `parse_workers` | Task 1 |
| Engine default/config/CLI precedence and positive resolved worker count | Task 2 |
| Engine call to parser worker-count-aware parse-all interface | Task 2 consumes existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` |
| Single-repo and multi-repo engine worker-count wiring | Task 2 |
| `generate-json --parse-workers` flag exposure and flag-presence override detection | Task 3 |
| CLI single and `--multi` forwarding of parse-worker override | Task 3 |
| Normalized one-worker versus multi-worker output equivalence | Task 4 |
| Parser worker-pool internals, result ordering implementation, and race-detector parser validation | Out of scope; owned by existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` |
| User documentation and benchmark report | Out of scope; owned by downstream rollout scope `architecture-main-1-architecture-3` |

## Spec Compliance Matrix

| # | Requirement | Source | Task(s) | Status |
|---|-------------|--------|---------|--------|
| G-1 | Output must remain deterministic for the same repository state and configuration. | `specs/map-reduce.md:33-39` | Task 4 | Covered |
| G-2 | Parallel parsing must not share mutable Tree-sitter parser or tree instances across workers. | `specs/map-reduce.md:33-39` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| G-3 | Memory growth must be bounded by configurable worker count. | `specs/map-reduce.md:33-39` | Task 1, Task 2; parser enforcement by existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| G-4 | Preserve existing behavior when configured for one worker. | `specs/map-reduce.md:33-39` | Task 1, Task 2, Task 4 | Covered |
| G-5 | Feature must be measurable with wall time, CPU profile, and maximum RSS. | `specs/map-reduce.md:33-39` | Downstream rollout scope `architecture-main-1-architecture-3` | Covered |
| FT1-FR1 | Parse source files through a bounded worker pool when configured parse worker count is greater than one. | `specs/map-reduce.md:76-83` | Task 2; worker-pool internals by existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-FR2 | Preserve parsed result order as if files were parsed sequentially in input order. | `specs/map-reduce.md:76-83` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0`; Task 4 verifies output equivalence | Covered |
| FT1-FR3 | Collect per-file parse errors without aborting the full parse, matching current `ParseAll` behavior. | `specs/map-reduce.md:76-83` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-FR4 | Do not share `gotreesitter.Parser`, syntax tree, or tree cursor values across workers. | `specs/map-reduce.md:76-83` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-FR5 | Support one-worker mode that exercises the same logical behavior as the current sequential implementation. | `specs/map-reduce.md:76-83` | Task 2, Task 4; parser compatibility by existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-NFR1 | Parallel parsing must not introduce nondeterministic module ordering, dependency ordering, or JSON output differences for the same input. | `specs/map-reduce.md:84-89` | Task 4 | Covered |
| FT1-NFR2 | Worker pool implementation must avoid unbounded goroutine creation. | `specs/map-reduce.md:84-89` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-NFR3 | Worker pool implementation must avoid retaining file contents longer than needed for existing parse result contract. | `specs/map-reduce.md:84-89` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-AC1 | One-worker and multi-worker `generate-json` runs produce equivalent normalized index content except generation metadata. | `specs/map-reduce.md:90-96` | Task 3, Task 4 | Covered |
| FT1-AC2 | Parallel parsing collects parse errors and keeps them non-fatal as in the sequential path. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-AC3 | Repeated runs against the same fixture and worker count produce identical normalized outputs. | `specs/map-reduce.md:90-96` | Task 4 | Covered |
| FT1-AC4 | Race detector reports no data races for parser tests exercising parallel parsing. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT2-FR1 | Provide a way to configure parse worker count for `generate-json`. | `specs/map-reduce.md:127-133` | Task 1, Task 2, Task 3 | Covered |
| FT2-FR2 | Treat worker count one as sequential compatibility mode. | `specs/map-reduce.md:127-133` | Task 1, Task 2, Task 4 | Covered |
| FT2-FR3 | Reject invalid worker counts less than one with a clear error. | `specs/map-reduce.md:127-133` | Task 1, Task 2, Task 3 | Covered |
| FT2-FR4 | Document worker-count control and memory tradeoff. | `specs/map-reduce.md:127-133` | Downstream rollout scope `architecture-main-1-architecture-3` | Covered |
| FT2-NFR1 | Default worker count must be conservative enough to avoid severe memory spikes on large repositories. | `specs/map-reduce.md:134-139` | Task 1, Task 2 | Covered |
| FT2-NFR2 | Configuration must be compatible with existing `.stacklitrc.json` files. | `specs/map-reduce.md:134-139` | Task 1 | Covered |
| FT2-NFR3 | Benchmark evidence must compare worker counts 1, 2, 4, and 8 before changing the default above one. | `specs/map-reduce.md:134-139` | Downstream rollout scope `architecture-main-1-architecture-3` | Covered |
| FT2-AC1 | `.stacklitrc.json` parse-worker configuration controls parser worker count. | `specs/map-reduce.md:140-145` | Task 1, Task 2, Task 4 | Covered |
| FT2-AC2 | Invalid configured worker count exits with clear configuration error. | `specs/map-reduce.md:140-145` | Task 1, Task 2, Task 3 | Covered |
| FT2-AC3 | Benchmark results report wall time and max RSS for worker counts 1, 2, 4, and 8. | `specs/map-reduce.md:140-145` | Downstream rollout scope `architecture-main-1-architecture-3` | Covered |
| FT2-AC4 | Users can identify how to enable/disable parallel parsing and understand higher worker counts may increase memory use. | `specs/map-reduce.md:140-145` | Downstream rollout scope `architecture-main-1-architecture-3` | Covered |
| ARCH-1 | Config exposes top-level `parse_workers`, default `1`, and validation for explicit values less than one. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:51-64` | Task 1 | Covered |
| ARCH-2 | `generate-json` exposes `--parse-workers` and uses flag presence for override detection. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:65-77` | Task 3 | Covered |
| ARCH-3 | Engine resolves one positive worker count from CLI/config/default and passes it to parser before reduce work. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:78-91` | Task 2 | Covered |
| ARCH-4 | Parser map stage receives ordered paths and exactly one resolved positive worker count. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:92-103` | Task 2 consumes existing parser task | Covered |
| ARCH-5 | Tests prove default/config/CLI precedence, invalid-value rejection, single/multi wiring, and normalized output equivalence. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:104-117` | Task 1, Task 2, Task 3, Task 4 | Covered |
| ARCH-6 | `RunMulti` forwards a CLI override to every `Run` call while absent CLI override lets each repo resolve its own config/default. | `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md:146-152` | Task 2, Task 3 | Covered |
| E2E | e2e test coverage for new behavior | Cross-cutting | Task 3 exercises `generate-json` command behavior; Task 4 exercises full engine single/multi output equivalence. | Covered |
| DOC | Documentation updates for changed behavior | Cross-cutting | N/A: user-facing documentation and benchmark report are explicitly out of scope for this plan and owned by downstream rollout scope `architecture-main-1-architecture-3`. | N/A |
