# Architecture Plan: Rollout Documentation and Benchmark Evidence

Status: draft

## Goal

Define the documentation and benchmark-evidence boundary for map-reduce parsing rollout without changing parser, config, CLI, engine, graph, or renderer behavior.

## Context

The parent map-reduce architecture splits runtime implementation from rollout evidence. The parser scope owns bounded parallel parsing. The worker-count control scope owns `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, CLI-over-config precedence, and engine-to-parser wiring. This scope consumes those implemented controls to update user documentation and record benchmark evidence required before any future default above one is considered.

### References

- Goal spec: `specs/map-reduce.md`
- Parent tasks: `architecture-main-1`, `architecture-main-1-architecture-2`
- Parent architecture: `specs/arch-plan/map-reduce/20260531-012947-architecture-main-1.md`
- Worker-count architecture: `specs/arch-plan/map-reduce/20260531-073048-architecture-main-1-architecture-2.md`
- Blackboard: `architecture-main-1` output summary, `architecture-main-1-architecture-2` output summary, `architecture-main-1-architecture-2-code-planning-0`
- Codebase and docs: `README.md`, `USAGE.md`, `internal/config/config.go`, `internal/cli/generate_json.go`, `internal/engine/engine.go`

### Constraints

- This scope owns only README/USAGE documentation and repo-tracked benchmark evidence under `specs/benchmarks/map-reduce/`.
- Runtime behavior remains read-only for this scope: no parser, config, CLI, engine, graph, renderer, schema, or generated index behavior changes.
- Documentation must describe both durable `.stacklitrc.json` control and command-local `generate-json --parse-workers` control after the worker-count control scope implements them.
- Documentation must state the memory tradeoff: higher worker counts can reduce wall time but may increase maximum RSS because more parser work can be live concurrently.
- Benchmark evidence must compare worker counts 1, 2, 4, and 8 on Omni or an equivalent large mixed-language repository at a fixed repository state.
- Benchmark evidence must include wall time, maximum RSS, and CPU profile evidence for every compared worker count.
- The default worker count remains `1` in this goal; benchmark evidence is a rollout input, not permission for this scope to raise the default.
- The canonical validation commands and profile file paths are fixed by the assigned task.
- A repo-wide bootstrap-precommit task already exists and has merged, so this architecture emits no bootstrap-precommit output entry.

### Assumptions

- **ASM-RD-001**: The downstream runtime control task will provide the documented `parse_workers` config key and `--parse-workers` CLI flag before this scope's coding task runs. *Why*: this task depends on `architecture-main-1-architecture-2-code-planning-0`, and the parent architecture orders rollout documentation after worker-count control. Confidence: HIGH.
- **ASM-RD-002**: CPU profiles can be collected as benchmark execution artifacts without adding product profiling behavior. *Why*: the goal spec cites profiling artifacts as evidence and this scope is explicitly read-only for runtime code. Confidence: MEDIUM.
- **ASM-RD-003**: A markdown report plus retained per-worker CPU profile files is the reviewable benchmark artifact shape. *Why*: the task requires a repo-tracked benchmark report and canonical validation runs `go tool pprof -top` against `specs/benchmarks/map-reduce/worker-{1,2,4,8}.cpu.pprof`. Confidence: HIGH.

### Open Questions

- None for this scope. The remaining profiling capture mechanics are code-planning details as long as the committed evidence satisfies the fixed artifact paths and validation commands without runtime code changes.

---

## Components

### User Documentation (`README.md`, `USAGE.md`)

**Responsibility:** Explain how users configure parse worker count and how to reason about the latency/memory tradeoff.

**Boundaries:**
- Exposes: concise README command/config references and complete USAGE reference material for `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, default `1`, CLI-over-config precedence, invalid values, and enable/disable guidance.
- Depends on: worker-count behavior from the runtime control scope.

**Key decisions:**
- Place the short control summary near existing `generate-json` and configuration documentation. Rationale: users currently find generation commands and `.stacklitrc.json` examples there.
- Keep the complete explanation in `USAGE.md` and the README concise. Rationale: README is a quick-start surface, while USAGE is the detailed command/config reference.
- Document disabling parallel parsing as worker count `1`. Rationale: the spec defines one-worker mode as sequential compatibility mode and the parent architecture keeps default `1`.
- Document memory risk alongside examples, not only in the benchmark report. Rationale: AC-002-4 requires users to understand that higher worker counts may increase memory use when reading documentation.

### Benchmark Evidence (`specs/benchmarks/map-reduce/`)

**Responsibility:** Preserve benchmark provenance and measurement evidence for the rollout decision.

**Boundaries:**
- Exposes: a repo-tracked markdown benchmark report and per-worker CPU profile artifacts at `specs/benchmarks/map-reduce/worker-1.cpu.pprof`, `worker-2.cpu.pprof`, `worker-4.cpu.pprof`, and `worker-8.cpu.pprof`.
- Depends on: implemented worker-count controls, the `stacklit` binary used by canonical benchmark commands, `/usr/bin/time -v`, `go tool pprof`, and a fixed large mixed-language repository state.

**Key decisions:**
- Store benchmark artifacts under `specs/benchmarks/map-reduce/`. Rationale: benchmark results are specification evidence and should be reviewable beside the goal spec, not hidden in `/tmp`.
- Record repository identity and fixed state in the report. Rationale: AC-002-3 requires all worker counts to be benchmarked on the same repository state.
- Record command lines, wall time, maximum RSS, output paths, and CPU-profile top evidence for every worker count. Rationale: NFR-000-5 requires wall time, CPU profile, and maximum RSS; AC-002-3 requires wall time and max RSS.
- Keep generated JSON outputs in `/tmp` for validation runs and do not commit them. Rationale: canonical benchmark commands write `/tmp/stacklit-workers-{1,2,4,8}.json`, and the repo-tracked artifact is the evidence report plus CPU profile files.

---

## Interfaces

### Runtime Worker Controls -> User Documentation

**Contract:** The implemented runtime control surface provides `.stacklitrc.json` `parse_workers`, `generate-json --parse-workers`, default `1`, CLI-over-config precedence, and clear rejection of values less than one.

**Direction:** Documentation consumes the implemented behavior and presents it to users.

**Invariants:** Documentation must not describe behavior that is absent from the runtime control task; examples must keep `1` as the compatibility/disable value; docs must not imply an automatic default above one.

### Runtime Worker Controls -> Benchmark Evidence

**Contract:** Benchmark execution invokes the implemented `generate-json --parse-workers N` control for N in 1, 2, 4, and 8, writing JSON outputs to the canonical `/tmp/stacklit-workers-N.json` paths.

**Direction:** Benchmark evidence consumes the CLI and engine behavior; it does not feed behavior back into the runtime path.

**Invariants:** All worker counts use the same repository state, same Stacklit build, and same output generation mode; benchmark collection must not change parser, config, CLI, engine, graph, or renderer behavior.

### Benchmark Evidence -> Reviewer Validation

**Contract:** The report records wall time and maximum RSS from `/usr/bin/time -v`, and the per-worker profile files are readable by `go tool pprof -top specs/benchmarks/map-reduce/worker-N.cpu.pprof`.

**Direction:** Reviewers validate committed benchmark evidence with the canonical benchmark and pprof commands.

**Invariants:** CPU profile evidence exists for workers 1, 2, 4, and 8; the report makes the fixed repository state explicit; benchmark JSON outputs are validation artifacts under `/tmp`, not committed files.

---

## Data Flow

```text
Implemented worker controls
  -> README/USAGE document config key, CLI flag, precedence, default, invalid values, memory tradeoff

Fixed large repository state + stacklit binary
  -> /usr/bin/time -v stacklit generate-json --parse-workers N -o /tmp/stacklit-workers-N.json
  -> benchmark report records wall time and maximum RSS for N in 1, 2, 4, 8
  -> CPU profile capture creates specs/benchmarks/map-reduce/worker-N.cpu.pprof
  -> go tool pprof -top verifies profile evidence for N in 1, 2, 4, 8
```

The docs and benchmark artifacts are consumers of runtime behavior. They do not introduce a new runtime data path.

---

## Cross-Cutting Concerns

| Concern | Approach |
|---------|----------|
| Error handling | Documentation describes invalid worker counts less than one as clear errors. Benchmark report records failures explicitly if a run cannot complete, but the code-planning task should not submit until all required runs and pprof validations succeed. |
| Observability | Benchmark evidence records external observations: `/usr/bin/time -v` wall time/max RSS and `go tool pprof -top` CPU profile evidence. No product telemetry or logging is added. |
| Configuration | Documentation covers `.stacklitrc.json` `parse_workers` and `generate-json --parse-workers`, including CLI-over-config precedence and default `1`. |
| Determinism | Benchmark report pins the repository state so worker-count comparisons are meaningful; JSON outputs stay in `/tmp` as validation artifacts. |
| Memory bounds | Documentation explains worker count as the concurrency/memory bound and warns that higher counts can increase maximum RSS. |
| Testing | Scope validation uses `go test ./...`, canonical benchmark commands for worker counts 1, 2, 4, 8, and `go tool pprof -top` against committed CPU profiles. |
| Security | This scope reads source repositories and writes benchmark artifacts only; it does not read credential files, execute user-provided scripts, add network calls, or weaken auth/authz. |

---

## Decomposition

Each scope becomes a code-planning child task. No bootstrap-precommit output entry is emitted because the repo-wide bootstrap-precommit task already exists and has merged.

### Scope 0: Rollout Documentation and Benchmark Evidence

**Component(s):** User documentation and benchmark evidence artifacts.

**Boundary:** Owns README/USAGE updates and a repo-tracked benchmark report with wall time, CPU profile evidence, and max RSS for worker counts 1, 2, 4, and 8. Does not change parser, config, CLI, engine, graph, or renderer behavior.

**Done when:** Documentation explains `.stacklitrc.json` and CLI worker controls plus memory tradeoffs, and benchmark evidence reports wall time, CPU profile evidence, and max RSS for worker counts 1, 2, 4, and 8 on Omni or an equivalent large mixed-language repository at a fixed repository state.

**Depends on:** Existing task `architecture-main-1-architecture-2-code-planning-0` for the runtime worker-control plan.

### Spec Coverage

| Spec Requirement | Scope |
|------------------|-------|
| NFR-000-5 feature measurable with wall time, CPU profile, and maximum RSS | Scope 0 benchmark report and CPU profile artifacts |
| FR-002-4 document worker-count control and memory tradeoff | Scope 0 README/USAGE updates |
| NFR-002-3 benchmark worker counts 1, 2, 4, and 8 before changing default above one | Scope 0 benchmark report |
| AC-002-3 report wall time and max RSS for worker counts 1, 2, 4, and 8 on same repository state | Scope 0 benchmark report |
| AC-002-4 users can identify how to enable/disable parallel parsing and understand higher worker counts may increase memory use | Scope 0 README/USAGE updates |
| NFR-002-1 default conservative enough to avoid severe memory spikes | Scope 0 documents default `1` and treats benchmark evidence as future rollout input; runtime default owned by worker-count control scope |
| FR-002-1 through FR-002-3, AC-002-1, AC-002-2 | Consumed from upstream worker-count control scope, not implemented here |
| FT-001 parser requirements, NFR-000-1 through NFR-000-4, AC-001-1 through AC-001-4 | Covered by upstream parser and worker-count control scopes, not implemented here |

### Shared-File Audit

| File or module | Writer scope | Readers |
|----------------|--------------|---------|
| `README.md` | Scope 0 | none in this decomposition |
| `USAGE.md` | Scope 0 | none in this decomposition |
| `specs/benchmarks/map-reduce/` | Scope 0 | reviewer validation and future rollout-default decisions |
| `internal/config/config.go` and config tests | Upstream worker-count control scope | Scope 0 reads behavior only |
| `internal/cli/generate_json.go` and CLI tests | Upstream worker-count control scope | Scope 0 reads behavior only |
| `internal/engine/engine.go` and engine tests | Upstream worker-count control scope | Scope 0 reads behavior only |
| `internal/parser/` | Upstream parser scope | Scope 0 reads behavior only |
| `.pre-commit-config.yaml` and project-owned pre-commit tooling manifest | Bootstrap-precommit task | Scope 0 consumes quality gate only |

### Dependency Order

Existing worker-count control code-planning task -> Scope 0.
