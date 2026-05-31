# Architecture Plan: Worker Count Control and Engine Wiring

Status: draft

## Goal

Define where `generate-json` resolves parse worker count, how the resolved count crosses from CLI/config into the parser map stage, and how integration tests prove one-worker and multi-worker scans produce equivalent normalized indexes.

## Context

The parent map-reduce architecture splits runtime work into a parser map-stage scope followed by this worker-count control scope. The parser scope owns the worker-pool internals and a worker-count-aware parser interface; this scope owns `.stacklitrc.json` support, `generate-json --parse-workers`, engine option wiring, single-repo and multi-repo invocation, and output equivalence tests. Existing `engine.Run` loads `.stacklitrc.json`, walks files, calls `parser.ParseAll(files)`, then performs graph/index assembly as the reduce phase. Existing `RunMulti` reads repo paths and calls `Run` for each repo.

### References

- Goal spec: `specs/map-reduce.md`
- Parent tasks: `architecture-main-1`, `architecture-main-1-architecture-1`
- Parent architecture: `specs/arch-plan/map-reduce/20260531-012947-architecture-main-1.md`
- Parser architecture: `specs/arch-plan/map-reduce/20260531-072218-architecture-main-1-architecture-1.md`
- Bootstrap architecture: `specs/arch-plan/map-reduce/20260531-014127-architecture-main-1-architecture-0.md`
- Blackboard: `architecture-main-1-architecture-2` task JSON, `architecture-main-1-architecture-1` task JSON, active-task summary, `architecture-main-1-architecture-1-code-planning-0`, `architecture-main-1-architecture-0-code-planning-0-coding-0`
- Codebase: `internal/config/config.go`, `internal/config/config_test.go`, `internal/cli/generate_json.go`, `internal/cli/index_query_test.go`, `internal/engine/engine.go`, `internal/engine/engine_test.go`, `internal/parser/parser.go`
- Symbol index: SCIP lookup for `Config`, `DefaultConfig`, `Run`, `RunMulti`, `ParseAll`, and `newGenerateJSONCmd`

### Constraints

- This architecture owns only config, CLI, engine wiring, and integration tests for worker-count control.
- Parser worker-pool internals remain owned by `architecture-main-1-architecture-1-code-planning-0`.
- User documentation and benchmark reports remain owned by the dependent rollout scope.
- The `stacklit.json` and multi-index schemas do not change.
- Default parse worker count remains `1`.
- Worker counts less than one are invalid and must be rejected with a clear error before parse work starts.
- Existing `.stacklitrc.json` files without `parse_workers` remain valid.
- CLI `--parse-workers` takes precedence over `.stacklitrc.json` for both single-repo and multi-repo scans.
- Multi-repo scans use the same resolved worker-count contract as single-repo scans; there is no separate multi-only parser path.
- A repo-wide bootstrap-precommit coding task is already in flight, so this architecture emits no bootstrap-precommit output entry.

### Assumptions

- **ASM-WC-001**: The parser scope will expose a worker-count-aware parse-all entry point that accepts an ordered path slice plus a positive worker count. *Why*: `architecture-main-1-architecture-1` owns that interface and this task is downstream of it. Confidence: HIGH.
- **ASM-WC-002**: Engine options can use `0` as the internal "no CLI override supplied" sentinel while config/default resolution always produces a positive count. *Why*: valid user worker counts start at `1`, and existing optional output overrides already use zero-value option fields for absence. Confidence: HIGH.
- **ASM-WC-003**: Normalizing generation metadata is sufficient for equivalence tests because parser result ordering and reduce semantics should be identical across worker counts. *Why*: parent scope preserves input-order parser results and this scope does not change graph or schema assembly. Confidence: MEDIUM.

### Open Questions

- None for this scope. The parent architecture resolves the goal spec's worker-count open questions as: default `1`, `.stacklitrc.json` plus CLI flag controls, and CLI-over-config precedence.

---

## Components

### Worker Count Configuration (`internal/config/`)

**Responsibility:** Represent `.stacklitrc.json` `parse_workers`, apply the conservative default, and expose validation for explicit invalid values.

**Boundaries:**
- Exposes: `Config.ParseWorkers` or equivalent scan setting with JSON key `parse_workers`, default value `1`, and a validation path that reports explicit values less than one.
- Depends on: `.stacklitrc.json` data loaded from a repository root and existing config defaults.

**Key decisions:**
- Keep `parse_workers` as a top-level `.stacklitrc.json` key. Rationale: existing scan-related controls such as `max_depth`, `max_modules`, and `max_exports` are top-level config fields.
- Default omitted or zero-value worker count to `1`. Rationale: one worker preserves current behavior and satisfies the conservative rollout requirement.
- Do not silently accept explicit negative worker counts. Rationale: the goal spec requires invalid configured worker counts to exit with a clear configuration error.
- Preserve compatibility for config files that omit the new key. Rationale: existing user config files must remain valid without edits.

### `generate-json` CLI (`internal/cli/generate_json.go`)

**Responsibility:** Expose the `--parse-workers` command override and pass override intent to engine for both single-repo and multi-repo scans.

**Boundaries:**
- Exposes: Cobra flag `--parse-workers` on `generate-json`.
- Depends on: Cobra flag parsing and engine `Options` / `MultiOptions`.

**Key decisions:**
- Use flag presence, not the flag's numeric zero value, to distinguish "no CLI override" from an invalid override. Rationale: `0` is invalid when supplied but useful as an internal absence sentinel.
- Pass the same override field to `engine.Run` and `engine.RunMulti`. Rationale: `generate-json --multi` should not fork worker-count precedence or parsing behavior.
- Keep CLI validation aligned with engine validation. Rationale: direct engine tests and CLI tests should observe the same invalid-count behavior.

### Engine Worker-Count Resolution (`internal/engine/`)

**Responsibility:** Resolve one positive parse worker count from default, repo config, and optional CLI override, then pass it to the parser map stage before graph/reduce work begins.

**Boundaries:**
- Exposes: `Run` and `RunMulti` behavior with optional parse-worker override fields in their option structs.
- Depends on: config loader/validation, walker output, and the parser worker-count-aware parse-all interface.

**Key decisions:**
- Resolve worker count inside `Run`, after loading the repo's config and before `walker.Walk`/parser invocation. Rationale: invalid worker-count configuration should fail before expensive scan work, and each repo in a multi scan has its own config file.
- Let CLI override replace repo config when supplied. Rationale: benchmark and one-off tuning commands need command-local precedence.
- Treat worker-count configuration errors as fatal for `generate-json`, including multi-repo scans. Rationale: "invalid configured worker count" is a configuration error, not a recoverable per-file parse warning.
- Keep graph building, metadata assembly, insights application, Merkle computation, and rendering unchanged. Rationale: the spec targets parser map-stage scheduling, not reduce semantics or output schema.

### Parser Map Stage Interface (`internal/parser/`)

**Responsibility:** Provide the worker-count-aware parsing boundary consumed by engine.

**Boundaries:**
- Exposes: an ordered parse-all interface that accepts positive worker count and returns the same logical result shape as current `ParseAll`.
- Depends on: implementation from the parser scope.

**Key decisions:**
- Engine passes only a resolved positive integer to parser. Rationale: parser should not know where user configuration came from.
- This scope does not change parser internals. Rationale: parser worker-pool behavior, result ordering, error collection, and Tree-sitter isolation are owned by the upstream parser code-planning task.

### Worker Count Integration Tests (`internal/config/config_test.go`, `internal/cli/index_query_test.go`, `internal/engine/engine_test.go`)

**Responsibility:** Prove default/config/CLI precedence, invalid-value rejection, engine-to-parser wiring in single and multi scans, and normalized output equivalence.

**Boundaries:**
- Exposes: package tests under `internal/config`, `internal/cli`, and `internal/engine`.
- Depends on: temporary fixture repositories, existing command constructors, and existing renderer/schema behavior.

**Key decisions:**
- Put config parsing and invalid config tests in `internal/config`. Rationale: config owns schema compatibility and validation semantics.
- Put flag exposure and CLI override tests in `internal/cli`. Rationale: CLI owns Cobra surface area and override detection.
- Put single-repo and multi-repo equivalence tests in `internal/engine`. Rationale: engine owns the full scan path from config resolution through parser invocation and output assembly.
- Normalize expected generation metadata before comparing indexes. Rationale: timestamps are expected to differ between runs and should not mask semantic equivalence.

---

## Interfaces

### Config -> Engine Worker-Count Resolution

**Contract:** Config provides a loaded config value whose parse worker setting resolves to a positive default of `1` when omitted, and reports an error when an explicit configured value is less than one.

**Direction:** `engine.Run` loads config for the scan root and asks for or computes the resolved config-sourced worker count.

**Invariants:** Existing config files without `parse_workers` remain accepted; invalid explicit worker counts fail before parser invocation; config does not know about CLI precedence.

### CLI -> Engine Worker-Count Resolution

**Contract:** `generate-json --parse-workers N` passes an optional override into `engine.Options` or `engine.MultiOptions`; absence of the flag leaves engine to use config/default resolution.

**Direction:** CLI calls engine after inspecting Cobra flag presence.

**Invariants:** CLI override takes precedence over `.stacklitrc.json`; supplied values less than one are rejected clearly; `--multi` forwards the same override to every repo scan.

### Engine Worker-Count Resolution -> Parser Map Stage Interface

**Contract:** Engine passes the ordered file path slice and exactly one resolved positive worker count to the parser map stage. Parser returns successful `[]*parser.FileInfo` and `[]error` in the same logical shape consumed today by graph building.

**Direction:** `engine.Run` calls parser after walking files and before `graph.Build`.

**Invariants:** Worker count one preserves compatibility; multi-worker parsing changes scheduling only; parser results and errors remain input-order deterministic for downstream reduce behavior.

### RunMulti -> Run

**Contract:** `RunMulti` reads repo paths, then calls `Run` for each repo with the same CLI-sourced parse-worker override and per-repo config resolution.

**Direction:** `RunMulti` orchestrates repeated `Run` calls.

**Invariants:** A CLI override applies uniformly across repos; absent CLI override lets each repo's `.stacklitrc.json` or default choose the count; invalid worker-count configuration is a clear failure rather than silent fallback.

### Integration Tests -> Runtime Pipeline

**Contract:** Tests exercise public package boundaries (`config.Load` or validation helpers, Cobra command execution, `engine.Run`, and `engine.RunMulti`) rather than reaching into parser worker internals.

**Direction:** Tests call package APIs and compare observable config values, command behavior, errors, and normalized output files or indexes.

**Invariants:** Equivalence comparisons ignore generation metadata only; semantic index content, module/dependency data, and multi-repo summaries must match between one-worker and multi-worker scans.

---

## Data Flow

```text
generate-json
  -> Cobra parses optional --parse-workers
  -> engine.Run or engine.RunMulti receives optional override
  -> engine.Run loads .stacklitrc.json for the scan root
  -> worker-count resolution chooses CLI override, else config parse_workers, else default 1
  -> invalid values fail with a clear configuration/flag error before parsing
  -> walker.Walk produces ordered file paths
  -> parser worker-count-aware map stage receives ordered paths + resolved positive count
  -> graph.Build and assembleIndex reduce parser results without schema changes
  -> renderer.WriteJSON or RunMulti multi-index output
```

For `generate-json --multi`, the CLI override enters `RunMulti` once and is forwarded to each `Run` call. Without a CLI override, each repo's own `.stacklitrc.json` can configure `parse_workers`.

---

## Cross-Cutting Concerns

| Concern | Approach |
|---------|----------|
| Error handling | Config and CLI invalid worker counts fail before parser work. Multi-repo scans treat worker-count config failures as configuration errors, not ordinary parse warnings. |
| Observability | Existing quiet/non-quiet output remains unchanged; this scope adds no runtime metrics or logs. Benchmark reporting is deferred to the rollout scope. |
| Configuration | `.stacklitrc.json` owns durable `parse_workers`; `generate-json --parse-workers` owns command-local override; default remains `1`. |
| Determinism | Engine keeps the reduce phase sequential and compares normalized indexes in tests; parser scope owns input-order result preservation. |
| Memory bounds | Engine passes exactly one positive count to parser; parser scope bounds active parse workers by that count. |
| Testing | Config tests cover schema/default/invalid config; CLI tests cover flag presence and override behavior; engine tests cover single/multi wiring and normalized output equivalence. |
| Security | Worker count is a bounded positive integer; no new command execution, network access, credential-file reads, or schema fields are introduced. |

---

## Decomposition

Each scope becomes a code-planning child task. No bootstrap-precommit output entry is emitted because `architecture-main-1-architecture-0-code-planning-0-coding-0` is already active for the repo-wide pre-commit bootstrap.

### Scope 0: Worker Count Control and Engine Wiring

**Component(s):** Worker count configuration, `generate-json` CLI, engine worker-count resolution, parser map-stage interface consumption, and worker-count integration tests.

**Boundary:** Owns `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, CLI-over-config precedence, engine-to-parser worker-count wiring for single and multi repo scans, and output equivalence tests. Does not implement parser worker-pool internals or write user documentation/benchmark reports.

**Done when:** `generate-json` resolves worker count from default/config/CLI with invalid values rejected, passes the resolved count into the parser map stage for both single and multi scans, and integration tests prove worker count one and multiple workers produce equivalent normalized indexes for fixtures.

**Depends on:** Existing task `architecture-main-1-architecture-1-code-planning-0` for the parser worker-count-aware parse-all contract.

### Spec Coverage

| Spec Requirement | Scope |
|------------------|-------|
| NFR-000-1 deterministic output for same repository/configuration | Scope 0 tests normalized output equivalence; parser scope preserves input-order results. |
| NFR-000-3 memory growth bounded by configurable worker count | Scope 0 resolves and passes the configured bound to parser; parser scope enforces active worker bounds. |
| NFR-000-4 preserve existing behavior when configured for one worker | Scope 0 default/config/CLI resolution keeps one-worker mode and tests equivalence. |
| AC-001-1 one-worker and multiple-worker fixture outputs are equivalent except generation metadata | Scope 0 |
| AC-001-3 repeated runs normalized for generation metadata are identical | Scope 0 |
| FR-002-1 provide a way to configure parse worker count for `generate-json` | Scope 0 |
| FR-002-2 worker count one is sequential compatibility mode | Scope 0 consumes parser one-worker compatibility and keeps default `1`. |
| FR-002-3 reject invalid worker counts less than one with a clear error | Scope 0 |
| NFR-002-1 conservative default avoids severe memory spikes | Scope 0 keeps default `1`. |
| NFR-002-2 configuration compatible with existing `.stacklitrc.json` files | Scope 0 |
| AC-002-1 configured parse workers control parser worker count | Scope 0 |
| AC-002-2 invalid configured worker count exits with clear configuration error | Scope 0 |
| FR-001-1 through FR-001-5, NFR-000-2, NFR-001-2, NFR-001-3, AC-001-2, AC-001-4 | Covered by upstream parser scope, not this scope. |
| FR-002-4, NFR-000-5, NFR-002-3, AC-002-3, AC-002-4 | Covered by downstream rollout documentation and benchmark scope, not this scope. |

### Shared-File Audit

| File or module | Writer scope | Readers |
|----------------|--------------|---------|
| `internal/config/config.go` and `internal/config/config_test.go` | Scope 0 | Documentation/benchmark scope reads behavior |
| `internal/cli/generate_json.go` and `internal/cli/index_query_test.go` | Scope 0 | Documentation/benchmark scope reads behavior |
| `internal/engine/engine.go` and `internal/engine/engine_test.go` | Scope 0 | Documentation/benchmark scope reads behavior |
| `internal/parser/` | Upstream parser scope | Scope 0 consumes worker-count-aware parser interface only |
| `README.md`, `USAGE.md`, `specs/benchmarks/map-reduce/` | Downstream rollout scope | Not modified by Scope 0 |
| `.pre-commit-config.yaml` and project-owned pre-commit tooling manifest | Bootstrap-precommit task | Scope 0 consumes quality gate only |

### Dependency Order

Existing upstream parser code-planning task -> Scope 0.

The bootstrap-precommit task remains a repo-wide support dependency consumed by downstream implementation tasks through the broader map-reduce task graph. Scope 0 then feeds the dependent rollout documentation and benchmark architecture/code-planning scope by establishing the user-visible runtime controls that scope documents and measures.
