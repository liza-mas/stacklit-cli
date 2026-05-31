# Code Plan: Rollout Documentation and Benchmark Evidence

Task ID: `architecture-main-1-architecture-3-code-planning-0`
Agent ID: `code-planner-1`
Spec reference: `specs/map-reduce.md`
Architecture reference: `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md`

## Source Evidence

- `specs/map-reduce.md`: full goal spec read for global requirement boundaries.
- `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md`: assigned rollout documentation and benchmark-evidence architecture.
- `specs/plans/map-reduce/20260531-074008-architecture-main-1-architecture-2-code-planning-0.md`: upstream worker-count control plan read for config, CLI, engine, and equivalence-test boundaries.
- `architecture-main-1-architecture-2-code-planning-0` output summary: upstream tasks expose `.stacklitrc.json` `parse_workers`, engine worker-count resolution, `generate-json --parse-workers`, and output equivalence coverage.
- `README.md` and `USAGE.md`: existing user-facing command and configuration documentation surfaces for `generate-json` and `.stacklitrc.json`.
- `specs/benchmarks/map-reduce/`: no existing benchmark evidence files are present in this worktree; this scope creates that artifact area.

## Planning Decision

Create two coding tasks. User documentation and benchmark evidence are separate deliverables with different validation surfaces and no shared files, so they can run in parallel after the upstream runtime worker-control implementation exists. Both tasks are non-runtime tasks: they must not change parser, config, CLI, engine, graph, renderer, schema, or generated index behavior.

## Task 1: Update Worker-Control User Documentation

Desc: Update README and USAGE documentation for parse worker controls and memory tradeoffs.

Done when: `README.md` gives a concise `generate-json --parse-workers` and `.stacklitrc.json` `parse_workers` summary, `USAGE.md` gives the detailed command and configuration reference, both documents state default worker count `1`, explain that worker count `1` disables parallel parsing as sequential compatibility mode, describe CLI-over-config precedence, mention that values less than one are invalid, and warn that higher worker counts may reduce wall time while increasing maximum RSS.

Scope: May edit `README.md` and `USAGE.md`. Must not edit parser, config, CLI, engine, graph, renderer, schema, benchmark artifacts, generated JSON outputs, or test fixtures. TDD is not required because this is a documentation-only task.

Spec_ref: `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md`

Plan_ref: `specs/plans/map-reduce/20260531-075015-architecture-main-1-architecture-3-code-planning-0.md`

Validation:

- `go test ./...`

Depends on existing tasks: `architecture-main-1-architecture-2-code-planning-0-coding-2`.

### Implementation Notes

- Keep README concise and place the short summary near existing `generate-json` and configuration material.
- Put the complete reference in `USAGE.md` under the command reference and configuration sections.
- Do not imply that benchmark evidence changes the default above `1`; this goal keeps the conservative default.
- Describe the control that exists after the upstream CLI task: `.stacklitrc.json` `parse_workers` is durable project configuration, and `generate-json --parse-workers N` is a command-local override.

## Task 2: Record Worker-Count Benchmark Evidence

Desc: Record benchmark evidence for parse worker counts 1, 2, 4, and 8.

Done when: `specs/benchmarks/map-reduce/` contains a markdown benchmark report plus `worker-1.cpu.pprof`, `worker-2.cpu.pprof`, `worker-4.cpu.pprof`, and `worker-8.cpu.pprof`; the report identifies Omni or an equivalent large mixed-language repository and its fixed repository state, records the Stacklit build or commit used, records the exact benchmark commands for worker counts 1, 2, 4, and 8, reports wall time and maximum RSS from `/usr/bin/time -v` for each worker count, records CPU-profile top evidence for each worker count, keeps generated JSON outputs in `/tmp/stacklit-workers-{1,2,4,8}.json` instead of committing them, and all four committed CPU profile files are readable by `go tool pprof -top`.

Scope: May add files under `specs/benchmarks/map-reduce/`, including the benchmark report and the four `worker-{1,2,4,8}.cpu.pprof` files. Must not edit `README.md`, `USAGE.md`, parser, config, CLI, engine, graph, renderer, schema, generated JSON outputs, or product profiling behavior. TDD is not required because this task records benchmark evidence rather than changing product behavior.

Spec_ref: `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md`

Plan_ref: `specs/plans/map-reduce/20260531-075015-architecture-main-1-architecture-3-code-planning-0.md`

Validation:

- `go test ./...`
- `/usr/bin/time -v stacklit generate-json --parse-workers 1 -o /tmp/stacklit-workers-1.json`
- `/usr/bin/time -v stacklit generate-json --parse-workers 2 -o /tmp/stacklit-workers-2.json`
- `/usr/bin/time -v stacklit generate-json --parse-workers 4 -o /tmp/stacklit-workers-4.json`
- `/usr/bin/time -v stacklit generate-json --parse-workers 8 -o /tmp/stacklit-workers-8.json`
- `go tool pprof -top specs/benchmarks/map-reduce/worker-1.cpu.pprof`
- `go tool pprof -top specs/benchmarks/map-reduce/worker-2.cpu.pprof`
- `go tool pprof -top specs/benchmarks/map-reduce/worker-4.cpu.pprof`
- `go tool pprof -top specs/benchmarks/map-reduce/worker-8.cpu.pprof`

Depends on existing tasks: `architecture-main-1-architecture-2-code-planning-0-coding-2`.

### Implementation Notes

- Use the canonical `/usr/bin/time -v stacklit generate-json --parse-workers N -o /tmp/stacklit-workers-N.json` commands as the measurement commands; do not commit the generated JSON outputs.
- Commit the CPU profile files at the exact paths used by canonical validation.
- The report should include enough provenance to make the comparison reproducible: target repository path or identity, target repository commit or equivalent fixed state, Stacklit commit or build source, date/time, hardware or runner summary if available, and any relevant environment notes.
- CPU profile capture is benchmark evidence only. Do not add product profiling flags, telemetry, or runtime behavior to collect it.
- If any benchmark run or pprof validation cannot complete, do not submit partial evidence; mark the coding task blocked with the exact failing command and stderr.

## Shared-File Audit

| File or module | Task(s) | Dependency handling |
|----------------|---------|---------------------|
| `README.md` | Task 1 | Single writer in this plan. |
| `USAGE.md` | Task 1 | Single writer in this plan. |
| `specs/benchmarks/map-reduce/` | Task 2 | Single writer in this plan. |
| `internal/config/`, `internal/cli/`, `internal/engine/`, `internal/parser/` | Existing upstream implementation tasks | Read-only dependency for this plan; no writer in this plan. |
| Generated `/tmp/stacklit-workers-{1,2,4,8}.json` files | Task 2 validation artifacts | Not committed and not shared with Task 1. |

## Cross-Reference Audit

| Reference | Owner |
|-----------|-------|
| README and USAGE documentation for `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, default `1`, CLI-over-config precedence, invalid values, and memory tradeoffs | Task 1 |
| Repo-tracked benchmark report | Task 2 |
| CPU profile artifacts at `specs/benchmarks/map-reduce/worker-{1,2,4,8}.cpu.pprof` | Task 2 |
| Wall time and maximum RSS measurements from `/usr/bin/time -v` for worker counts 1, 2, 4, and 8 | Task 2 |
| Fixed large mixed-language repository state provenance | Task 2 |
| Parser worker-pool implementation | Out of scope; existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` |
| `.stacklitrc.json` `parse_workers` config implementation, engine resolution, and `generate-json --parse-workers` CLI implementation | Out of scope; existing upstream task chain ending at `architecture-main-1-architecture-2-code-planning-0-coding-2` |
| Normalized output equivalence tests | Out of scope; existing task `architecture-main-1-architecture-2-code-planning-0-coding-3` |

## Spec Compliance Matrix

| # | Requirement | Source | Task(s) | Status |
|---|-------------|--------|---------|--------|
| G-1 | Output must remain deterministic for the same repository state and configuration. | `specs/map-reduce.md:33-39` | Upstream parser and engine tasks; Task 2 records benchmark state provenance without changing output behavior. | Covered |
| G-2 | Parallel parsing must not share mutable Tree-sitter parser or tree instances across workers. | `specs/map-reduce.md:33-39` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| G-3 | Memory growth must be bounded by configurable worker count. | `specs/map-reduce.md:33-39` | Upstream worker-control tasks; Task 1 documents worker count as the memory bound and Task 2 records max RSS. | Covered |
| G-4 | Preserve existing behavior when configured for one worker. | `specs/map-reduce.md:33-39` | Upstream worker-control tasks; Task 1 documents worker count `1` compatibility mode and Task 2 benchmarks worker count `1`. | Covered |
| G-5 | Feature must be measurable with wall time, CPU profile, and maximum RSS. | `specs/map-reduce.md:33-39` | Task 2 | Covered |
| FT1-FR1 | Parse source files through a bounded worker pool when configured parse worker count is greater than one. | `specs/map-reduce.md:76-83` | Existing parser and worker-control tasks | Covered |
| FT1-FR2 | Preserve parsed result order as if files were parsed sequentially in input order. | `specs/map-reduce.md:76-83` | Existing parser task and upstream normalized output tests | Covered |
| FT1-FR3 | Collect per-file parse errors without aborting the full parse, matching current `ParseAll` behavior. | `specs/map-reduce.md:76-83` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-FR4 | Do not share `gotreesitter.Parser`, syntax tree, or tree cursor values across workers. | `specs/map-reduce.md:76-83` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-FR5 | Support one-worker mode that exercises the same logical behavior as the current sequential implementation. | `specs/map-reduce.md:76-83` | Existing parser and worker-control tasks; Task 1 documents it and Task 2 benchmarks it. | Covered |
| FT1-NFR1 | Parallel parsing must not introduce nondeterministic module ordering, dependency ordering, or JSON output differences for the same input. | `specs/map-reduce.md:84-89` | Existing task `architecture-main-1-architecture-2-code-planning-0-coding-3` | Covered |
| FT1-NFR2 | Worker pool implementation must avoid unbounded goroutine creation. | `specs/map-reduce.md:84-89` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-NFR3 | Worker pool implementation must avoid retaining file contents longer than needed for existing parse result contract. | `specs/map-reduce.md:84-89` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-AC1 | One-worker and multi-worker `generate-json` runs produce equivalent normalized index content except generation metadata. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-2-code-planning-0-coding-3` | Covered |
| FT1-AC2 | Parallel parsing collects parse errors and keeps them non-fatal as in the sequential path. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT1-AC3 | Repeated runs against the same fixture and worker count produce identical normalized outputs. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-2-code-planning-0-coding-3` | Covered |
| FT1-AC4 | Race detector reports no data races for parser tests exercising parallel parsing. | `specs/map-reduce.md:90-96` | Existing task `architecture-main-1-architecture-1-code-planning-0-coding-0` | Covered |
| FT2-FR1 | Provide a way to configure parse worker count for `generate-json`. | `specs/map-reduce.md:127-133` | Existing worker-control tasks; Task 1 documents the controls. | Covered |
| FT2-FR2 | Treat worker count one as sequential compatibility mode. | `specs/map-reduce.md:127-133` | Existing worker-control tasks; Task 1 documents it and Task 2 benchmarks it. | Covered |
| FT2-FR3 | Reject invalid worker counts less than one with a clear error. | `specs/map-reduce.md:127-133` | Existing worker-control tasks; Task 1 documents invalid values. | Covered |
| FT2-FR4 | Document worker-count control and memory tradeoff. | `specs/map-reduce.md:127-133` | Task 1 | Covered |
| FT2-NFR1 | Default worker count must be conservative enough to avoid severe memory spikes on large repositories. | `specs/map-reduce.md:134-139` | Existing config task keeps default `1`; Task 1 documents default `1`; Task 2 records evidence for future rollout decisions. | Covered |
| FT2-NFR2 | Configuration must be compatible with existing `.stacklitrc.json` files. | `specs/map-reduce.md:134-139` | Existing task `architecture-main-1-architecture-2-code-planning-0-coding-0`; Task 1 documents additive config. | Covered |
| FT2-NFR3 | Benchmark evidence must compare worker counts 1, 2, 4, and 8 before changing the default above one. | `specs/map-reduce.md:134-139` | Task 2 | Covered |
| FT2-AC1 | `.stacklitrc.json` parse-worker configuration controls parser worker count. | `specs/map-reduce.md:140-145` | Existing worker-control tasks; Task 1 documents the behavior. | Covered |
| FT2-AC2 | Invalid configured worker count exits with clear configuration error. | `specs/map-reduce.md:140-145` | Existing worker-control tasks; Task 1 documents the behavior. | Covered |
| FT2-AC3 | Benchmark results report wall time and max RSS for worker counts 1, 2, 4, and 8. | `specs/map-reduce.md:140-145` | Task 2 | Covered |
| FT2-AC4 | Users can identify how to enable/disable parallel parsing and understand higher worker counts may increase memory use. | `specs/map-reduce.md:140-145` | Task 1 | Covered |
| ARCH3-C1 | This scope owns only README/USAGE documentation and repo-tracked benchmark evidence under `specs/benchmarks/map-reduce/`. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 1, Task 2 | Covered |
| ARCH3-C2 | Runtime behavior remains read-only for this scope: no parser, config, CLI, engine, graph, renderer, schema, or generated index behavior changes. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 1, Task 2 | Covered |
| ARCH3-C3 | Documentation must describe durable `.stacklitrc.json` control and command-local `generate-json --parse-workers` control. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 1 | Covered |
| ARCH3-C4 | Documentation must state the memory tradeoff for higher worker counts. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 1 | Covered |
| ARCH3-C5 | Benchmark evidence must compare worker counts 1, 2, 4, and 8 on Omni or an equivalent large mixed-language repository at a fixed repository state. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 2 | Covered |
| ARCH3-C6 | Benchmark evidence must include wall time, maximum RSS, and CPU profile evidence for every compared worker count. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 2 | Covered |
| ARCH3-C7 | The default worker count remains `1` in this goal; benchmark evidence is not permission to raise it. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:22-33` | Task 1, Task 2 | Covered |
| ARCH3-I1 | Documentation covers config key, CLI flag, precedence, default, invalid values, and enable/disable guidance. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:48-60` | Task 1 | Covered |
| ARCH3-I2 | Benchmark report records repository state, commands, wall time, maximum RSS, output paths, and CPU-profile top evidence. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:62-74` | Task 2 | Covered |
| ARCH3-I3 | CPU profile files exist for workers 1, 2, 4, and 8 and are readable by `go tool pprof -top`. | `specs/arch-plan/map-reduce/20260531-074032-architecture-main-1-architecture-3.md:96-102` | Task 2 | Covered |
| E2E | e2e test coverage for new behavior | Cross-cutting | N/A: this plan does not change runtime behavior; upstream `architecture-main-1-architecture-2-code-planning-0-coding-3` owns end-to-end normalized output equivalence coverage for the worker-count behavior. | N/A |
| DOC | Documentation updates for changed behavior | Cross-cutting | Task 1 | Covered |
