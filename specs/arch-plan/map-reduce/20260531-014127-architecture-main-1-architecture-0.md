# Architecture Plan: Pre-Commit Bootstrap for Map-Reduce Work

Status: draft

## Goal

Define the support-only pre-commit bootstrap boundary required before downstream map-reduce implementation tasks run their shared quality gate.

## Context

The parent map-reduce architecture emits this task as Scope 0 because the integration branch lacks `.pre-commit-config.yaml` and there is no other active repo-wide `bootstrap-precommit` task outside this architecture task. The runtime map-reduce work remains owned by sibling architecture/code-planning scopes; this scope only establishes the repository quality gate those tasks will consume.

### References

- Goal spec: `specs/map-reduce.md`
- Parent tasks: `architecture-main-1`
- Parent architecture: `specs/arch-plan/map-reduce/20260531-012947-architecture-main-1.md`
- Codebase: `Makefile`, `go.mod`, `go.sum`
- Blackboard: `architecture-main-1-architecture-0` task JSON and active-task summary
- Git evidence: `integration:.pre-commit-config.yaml` is absent

### Constraints

- This scope does not touch map-reduce parser, engine, CLI, config, docs, or tests.
- The implementation may provision pre-commit only via a project-scoped mechanism already available in the repo or explicitly defined by this task/config.
- The implementation must not install OS packages or unrelated global tooling.
- If no authorized project-scoped path exists during code planning or implementation, the downstream task must mark BLOCKED rather than mutating host tooling.
- Hook categories must cover Go formatting/tests and general text hygiene for markdown, YAML, JSON, and trailing whitespace.
- Pre-commit must be recorded as a dev dependency in the selected project-owned tooling manifest or lockfile.
- The canonical validation command is `pre-commit run --all-files`.

### Assumptions

- **ASM-PC-001**: A Python-style `requirements-dev.txt` is an acceptable project-scoped tooling manifest for recording the pre-commit dev dependency. *Why*: the repository is a Go module with no existing dev-tool manifest, and this task explicitly owns any project-scoped tooling manifest required to run pre-commit. Confidence: MEDIUM.
- **ASM-PC-002**: Local pre-commit hooks may invoke Go toolchain commands directly from `.pre-commit-config.yaml`. *Why*: the repository already relies on Go commands through `go.mod` and `Makefile`, and the hook definitions themselves are repo-owned configuration. Confidence: HIGH.

### Open Questions

- None for architecture. If the implementation environment cannot install or execute pre-commit through the project-owned dev dependency manifest, the downstream task must mark BLOCKED with the exact failing command and stderr.

---

## Components

### Pre-Commit Configuration (`.pre-commit-config.yaml`)

**Responsibility:** Own the repo-level hook graph that downstream tasks run before submission.

**Boundaries:**
- Exposes: a `pre-commit run --all-files` gate that validates Go formatting/tests and text hygiene.
- Depends on: the Go toolchain already required by `go.mod`, the project-owned pre-commit tooling manifest, and pre-commit hook repositories or local hook commands declared in `.pre-commit-config.yaml`.

**Key decisions:**
- Use local hooks for Go checks: `gofmt` over Go files and `go test ./...` for package tests. Rationale: Go validation should stay tied to the repository's own Go module rather than a generic external formatter.
- Use standard pre-commit text hygiene hooks for trailing whitespace, end-of-file normalization, YAML syntax, JSON syntax, and markdown-oriented whitespace coverage. Rationale: these are repository-wide hygiene checks and do not encode map-reduce behavior.
- Keep hook selection unpinned at architecture level. Rationale: version pins are explicitly deferred out of this bootstrap architecture task.

### Project-Scoped Tooling Manifest (`requirements-dev.txt` or equivalent)

**Responsibility:** Record pre-commit as a dev dependency through a repo-owned file.

**Boundaries:**
- Exposes: a project-local declaration that `pre-commit` is required for development validation.
- Depends on: an authorized project-scoped provisioning path available to the downstream code task.

**Key decisions:**
- Prefer `requirements-dev.txt` unless code planning finds an already-present project-owned dev-tool manifest. Rationale: the current repository has `go.mod`, `go.sum`, and `Makefile`, but no existing Python, Node, or dedicated dev-tool dependency manifest.
- Do not introduce lockfiles or package-manager wrappers unless the chosen project-scoped mechanism requires them. Rationale: this scope exists only to bootstrap the quality gate, not to redesign repository tooling.

---

## Interfaces

### Downstream Tasks -> Pre-Commit Configuration

**Contract:** Downstream implementation tasks run `pre-commit run --all-files` and treat failures as actionable quality-gate failures.

**Direction:** Code tasks call the pre-commit executable after their changes and before submission.

**Invariants:** The command operates on repository files, may modify files only through formatter hooks, and must exit 0 for the bootstrap commit before this support scope is complete.

### Pre-Commit Configuration -> Go Toolchain

**Contract:** Local hooks execute Go formatting and tests against the module rooted at `go.mod`.

**Direction:** Pre-commit invokes Go commands from hook entries.

**Invariants:** Go formatting covers tracked Go source files; Go tests run at repository scope; hook definitions do not change runtime map-reduce behavior.

### Project-Scoped Tooling Manifest -> Pre-Commit Executable

**Contract:** The manifest records `pre-commit` as a development dependency so the executable is provisioned through project-owned tooling rather than OS/global mutation.

**Direction:** Developers or automation install/use the dependency according to the selected project-scoped mechanism, then invoke the canonical `pre-commit run --all-files` command.

**Invariants:** No OS packages or unrelated global tools are installed as part of this task.

---

## Data Flow

```text
developer or automation
  -> project-scoped tooling manifest records pre-commit
  -> pre-commit run --all-files
  -> .pre-commit-config.yaml
       -> local Go formatting hook
       -> local Go test hook
       -> text hygiene hooks for markdown, YAML, JSON, trailing whitespace, EOF
  -> exit 0 on the bootstrap commit
```

The data flow is repository validation only. It does not feed into parser, engine, CLI, config, graph, renderer, or documentation behavior.

---

## Cross-Cutting Concerns

| Concern | Approach |
|---------|----------|
| Error handling | Hook failures are reported by pre-commit; missing authorized provisioning path is a BLOCKED condition for the downstream task. |
| Observability | `pre-commit run --all-files` output is the validation evidence. |
| Configuration | `.pre-commit-config.yaml` owns hook behavior; the tooling manifest records the pre-commit dev dependency. |
| Testing | The bootstrap task validates itself by running `pre-commit run --all-files` against the bootstrap commit. |
| Security | No secrets or credential files are read; no OS packages or unrelated global tooling are installed. |
| Scope control | Runtime map-reduce packages and docs are excluded from this support task. |

---

## Decomposition

Each scope becomes a code-planning child task.

### Scope 0: Pre-Commit Bootstrap

**Component(s):** `.pre-commit-config.yaml` and project-scoped pre-commit tooling manifest.

**Boundary:** Owns `.pre-commit-config.yaml` and any project-scoped tooling manifest required to run pre-commit. This task may provision pre-commit only via a project-scoped mechanism already available in the repo or explicitly defined by the task/config. Do not install OS packages or unrelated global tooling. If no authorized project-scoped path exists, mark BLOCKED. Hook categories and concrete hooks are: local Go formatting with `gofmt`, local Go tests with `go test ./...`, and general text hygiene hooks for markdown-relevant trailing whitespace/end-of-file normalization plus YAML and JSON syntax checks. Does not touch map-reduce parser, engine, CLI, config, docs, or tests.

**Done when:** A project-scoped pre-commit configuration exists at `.pre-commit-config.yaml`, pre-commit is recorded as a dev dependency in `requirements-dev.txt` or the selected equivalent project-owned tooling manifest/lockfile, and after provisioning from that project-owned manifest the canonical `pre-commit run --all-files` command exits 0 against the bootstrap commit.

**Depends on:** none.

### Spec Coverage

| Spec Requirement | Scope |
|------------------|-------|
| Repository quality-gate bootstrap required by parent map-reduce decomposition | Scope 0 |
| Runtime map-reduce parsing, worker-count configuration, rollout docs, and benchmark evidence | Covered by sibling scopes from `architecture-main-1`, not by this support-only task |

### Shared-File Audit

| File or module | Writer scope | Readers |
|----------------|--------------|---------|
| `.pre-commit-config.yaml` | Scope 0 | All downstream code tasks as quality-gate users |
| `requirements-dev.txt` or selected equivalent tooling manifest/lockfile | Scope 0 | All downstream code tasks that need project-scoped pre-commit provisioning |

### Dependency Order

Scope 0 has no incoming dependencies. Sibling map-reduce implementation scopes should depend on the resulting bootstrap-precommit task before relying on repository pre-commit validation.
