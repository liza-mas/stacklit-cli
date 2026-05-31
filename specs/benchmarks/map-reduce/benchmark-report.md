# Map-Reduce Parse Worker Benchmark

Date: 2026-05-31T09:56:24Z

## Target Repository

- Repository: Omni, cloned from `/home/tangi/Workspace/omni`
- Benchmark clone: `/tmp/stacklit-benchmark-omni-coder-1-20260531T095624Z`
- Fixed state: `462588106cf285c017778073a22ac2e7e57376c2`
- Clone preparation:
  - `git clone --no-checkout /home/tangi/Workspace/omni /tmp/stacklit-benchmark-omni-coder-1-20260531T095624Z`
  - `git -C /tmp/stacklit-benchmark-omni-coder-1-20260531T095624Z checkout --detach 462588106cf285c017778073a22ac2e7e57376c2`
- Clean-state check after benchmark: `git -C /tmp/stacklit-benchmark-omni-coder-1-20260531T095624Z status --short --branch` reported `## HEAD (no branch)`.

## Stacklit Build

- Stacklit source commit: `ee2fa7c730d87d078e849c594fa944bb108b552c`
- Go version: `go version go1.26.0 linux/amd64`
- Binary path: `/tmp/stacklit-map-reduce-benchmark`
- Reported binary version: `stacklit version source:benchmark:ee2fa7c730d87d078e849c594fa944bb108b552c`
- Build command, run from `/tmp/stacklit-map-reduce-profile-main`:

```bash
GOSUMDB=off GOMODCACHE=/home/tangi/.cache/go-mod go build -buildvcs=false -ldflags "-X github.com/glincker/stacklit/internal/cli.Version=source:benchmark:ee2fa7c730d87d078e849c594fa944bb108b552c" -o /tmp/stacklit-map-reduce-benchmark .
```

The temporary build main delegates to `github.com/glincker/stacklit/internal/cli` and starts `runtime/pprof` only when `STACKLIT_BENCH_CPU_PROFILE` is set. No repository source files were changed to add profiling behavior. The timing commands below were run without that environment variable.

## Runner

- Host: `Linux tangi-t14 6.8.0-111-generic #111-Ubuntu SMP PREEMPT_DYNAMIC Sat Apr 11 23:16:02 UTC 2026 x86_64`
- CPUs reported by `nproc`: `8`
- Memory before report collection: `31Gi total`, `14Gi available`, `15Gi swap`

## Timing Results

All timing commands were run from `/tmp/stacklit-benchmark-omni-coder-1-20260531T095624Z`. Generated JSON outputs were kept in `/tmp/stacklit-workers-{1,2,4,8}.json` and are not committed.

| Parse workers | Command | Wall time | Maximum RSS |
|---:|---|---:|---:|
| 1 | `/usr/bin/time -v /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 1 -o /tmp/stacklit-workers-1.json` | `2:16.19` | `2555192 kbytes` |
| 2 | `/usr/bin/time -v /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 2 -o /tmp/stacklit-workers-2.json` | `1:44.53` | `4858316 kbytes` |
| 4 | `/usr/bin/time -v /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 4 -o /tmp/stacklit-workers-4.json` | `1:05.80` | `5296780 kbytes` |
| 8 | `/usr/bin/time -v /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 8 -o /tmp/stacklit-workers-8.json` | `1:10.14` | `7494828 kbytes` |

JSON output sizes observed in `/tmp`: each worker output was `46K`.

## CPU Profile Capture

CPU profiles were captured with the same `/tmp/stacklit-map-reduce-benchmark` binary from the same Omni clone and fixed commit. The only added environment variable selects the output profile path.

```bash
STACKLIT_BENCH_CPU_PROFILE=/home/tangi/Workspace/stacklit/.worktrees/architecture-main-1-architecture-3-code-planning-0-coding-1-replacement-0/specs/benchmarks/map-reduce/worker-1.cpu.pprof /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 1 -o /tmp/stacklit-workers-1.json
STACKLIT_BENCH_CPU_PROFILE=/home/tangi/Workspace/stacklit/.worktrees/architecture-main-1-architecture-3-code-planning-0-coding-1-replacement-0/specs/benchmarks/map-reduce/worker-2.cpu.pprof /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 2 -o /tmp/stacklit-workers-2.json
STACKLIT_BENCH_CPU_PROFILE=/home/tangi/Workspace/stacklit/.worktrees/architecture-main-1-architecture-3-code-planning-0-coding-1-replacement-0/specs/benchmarks/map-reduce/worker-4.cpu.pprof /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 4 -o /tmp/stacklit-workers-4.json
STACKLIT_BENCH_CPU_PROFILE=/home/tangi/Workspace/stacklit/.worktrees/architecture-main-1-architecture-3-code-planning-0-coding-1-replacement-0/specs/benchmarks/map-reduce/worker-8.cpu.pprof /tmp/stacklit-map-reduce-benchmark generate-json --parse-workers 8 -o /tmp/stacklit-workers-8.json
```

Profile files committed with this report:

- `specs/benchmarks/map-reduce/worker-1.cpu.pprof` (`94K`)
- `specs/benchmarks/map-reduce/worker-2.cpu.pprof` (`100K`)
- `specs/benchmarks/map-reduce/worker-4.cpu.pprof` (`121K`)
- `specs/benchmarks/map-reduce/worker-8.cpu.pprof` (`137K`)

## CPU Profile Top Evidence

### Worker 1

```text
File: stacklit-map-reduce-benchmark
Type: cpu
Duration: 134.44s, Total samples = 143.12s (106.46%)
Showing nodes accounting for 125.03s, 87.36% of 143.12s total
      flat  flat%   sum%        cum   cum%
    44.36s 30.99% 30.99%     48.68s 34.01%  github.com/odvcencio/gotreesitter.lookupNodeEquivCache
    23.47s 16.40% 47.39%     74.92s 52.35%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentFrontierWithScratch
     5.86s  4.09% 51.49%     86.16s 60.20%  github.com/odvcencio/gotreesitter.gssStacksEqualForLanguageWithScratch
     5.19s  3.63% 55.11%     80.11s 55.97%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentForLanguageWithScratch
     4.76s  3.33% 58.44%     12.57s  8.78%  runtime.scanObject
```

### Worker 2

```text
File: stacklit-map-reduce-benchmark
Type: cpu
Duration: 80.25s, Total samples = 163.68s (203.97%)
Showing nodes accounting for 142.94s, 87.33% of 163.68s total
      flat  flat%   sum%        cum   cum%
    49.26s 30.10% 30.10%     54.29s 33.17%  github.com/odvcencio/gotreesitter.lookupNodeEquivCache
    27.69s 16.92% 47.01%     85.64s 52.32%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentFrontierWithScratch
     8.11s  4.95% 51.97%     99.78s 60.96%  github.com/odvcencio/gotreesitter.gssStacksEqualForLanguageWithScratch
     5.96s  3.64% 55.61%     15.12s  9.24%  runtime.scanObject
     5.85s  3.57% 59.18%     91.52s 55.91%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentForLanguageWithScratch
```

### Worker 4

```text
File: stacklit-map-reduce-benchmark
Type: cpu
Duration: 67.93s, Total samples = 206.48s (303.94%)
Showing nodes accounting for 183.04s, 88.65% of 206.48s total
      flat  flat%   sum%        cum   cum%
    57.36s 27.78% 27.78%     63.21s 30.61%  github.com/odvcencio/gotreesitter.lookupNodeEquivCache
    33.72s 16.33% 44.11%    102.22s 49.51%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentFrontierWithScratch
    10.19s  4.94% 49.05%    119.27s 57.76%  github.com/odvcencio/gotreesitter.gssStacksEqualForLanguageWithScratch
     6.94s  3.36% 52.41%     17.79s  8.62%  runtime.scanObject
     6.69s  3.24% 55.65%    108.98s 52.78%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentForLanguageWithScratch
```

### Worker 8

```text
File: stacklit-map-reduce-benchmark
Type: cpu
Duration: 60.90s, Total samples = 223.03s (366.23%)
Showing nodes accounting for 196.50s, 88.10% of 223.03s total
      flat  flat%   sum%        cum   cum%
    59.07s 26.49% 26.49%     66.40s 29.77%  github.com/odvcencio/gotreesitter.lookupNodeEquivCache
    35.55s 15.94% 42.42%    110.13s 49.38%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentFrontierWithScratch
    11.71s  5.25% 47.68%    129.74s 58.17%  github.com/odvcencio/gotreesitter.gssStacksEqualForLanguageWithScratch
     7.52s  3.37% 51.05%    117.69s 52.77%  github.com/odvcencio/gotreesitter.stackEntryNodesEquivalentForLanguageWithScratch
     7.03s  3.15% 54.20%      7.08s  3.17%  github.com/odvcencio/gotreesitter.nodeEquivCacheIndex (inline)
```

## Validation Commands

```bash
go test ./...
go tool pprof -top specs/benchmarks/map-reduce/worker-1.cpu.pprof
go tool pprof -top specs/benchmarks/map-reduce/worker-2.cpu.pprof
go tool pprof -top specs/benchmarks/map-reduce/worker-4.cpu.pprof
go tool pprof -top specs/benchmarks/map-reduce/worker-8.cpu.pprof
```
