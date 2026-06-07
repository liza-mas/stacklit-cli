# Stacklit - Architecture Export

## Goal

Expose architectural structure discovered by Stacklit.

The export provides architectural context that complements symbol-level information.

## Philosophy

Stacklit describes structure.

It does not perform graph analytics.

It does not perform clustering.

It does not perform impact analysis.

## Inputs

Required:

- Stacklit index

## Artifact Metadata

Every export includes:

- `schema_version`: `"stacklit.architecture-export.v1"`
- `generator`: tool name and version string
- `generated_at`: UTC RFC3339 timestamp
- `inputs.stacklit_index.path`: caller-supplied Stacklit index path
- `inputs.stacklit_index.fingerprint`: stable fingerprint of the consumed index bytes

The export command reads only the supplied Stacklit index. It does not
regenerate Stacklit data and does not parse source files.

## Outputs

### Packages

For each package:

- identifier
- name
- description

In v1, a package is a normalized Stacklit module unless a future Stacklit schema
adds a distinct package concept.

### Components

For each component:

- identifier
- name
- description

In v1, a component is a normalized Stacklit module-level architectural unit.
If no separate component grouping is present in the Stacklit index, components
are one-to-one with packages.

### Membership

Mappings:

- symbol -> package
- symbol -> component

Stacklit does not currently provide stable SCIP symbols. V1 membership is
therefore path-based:

- `path -> package`
- `path -> component`

Symbol membership is derived by consumers by joining SCIP symbol document paths
to these path mappings.

### Relationships

Examples:

- package dependency
- component dependency
- ownership relationship

### Entry Points

Optional architectural entry points:

- controllers
- public APIs
- commands
- services

Each entry point records:

- path
- kind
- entry point source
- entry point confidence

Supported v1 `entry_point_source` values:

- `parser`
- `framework_pattern`
- `config`
- `heuristic`

`entry_point_confidence` is a number from `0.0` to `1.0` describing detection
confidence. Low-confidence entry points may still be useful for summaries, but
consumers must not treat them as guaranteed runtime entry points.

## JSON Schema

The v1 JSON shape is:

```json
{
  "schema_version": "stacklit.architecture-export.v1",
  "generator": {
    "name": "stacklit",
    "version": "..."
  },
  "generated_at": "2026-06-07T00:00:00Z",
  "inputs": {
    "stacklit_index": {
      "path": "stacklit.json",
      "fingerprint": "sha256:..."
    }
  },
  "packages": [
    {
      "id": "internal/commands",
      "name": "internal/commands",
      "description": "User-facing command handlers"
    }
  ],
  "components": [
    {
      "id": "internal/commands",
      "name": "internal/commands",
      "description": "User-facing command handlers"
    }
  ],
  "membership": [
    {
      "path": "internal/commands/root.go",
      "package": "internal/commands",
      "component": "internal/commands"
    }
  ],
  "relationships": [
    {
      "source": "internal/commands",
      "target": "internal/ops",
      "type": "package_dependency"
    }
  ],
  "entry_points": [
    {
      "path": "cmd/liza/main.go",
      "kind": "command",
      "entry_point_source": "parser",
      "entry_point_confidence": 0.8
    }
  ]
}
```

Arrays are present even when empty. Unknown optional facts are omitted, not
guessed.

## Identity

Package and component identifiers are repository-relative module paths from the
Stacklit index.

Path membership uses repository-relative file paths from the Stacklit index.
Consumers that join this export with SCIP data must normalize both sides to the
same slash-separated repository-relative path form before joining.

## Error Behavior

If the Stacklit index cannot be read or decoded, the command fails and emits no
partial artifact.

If Stacklit cannot classify a module, the export keeps the module with its
identifier and an empty or generic description. Missing optional entry points or
relationships are represented by empty arrays.

## Output Formats

### JSON

Human-readable.

### Binary

Future / out of scope for v1.

No binary format is part of this contract until a concrete consumer requires
one.

## Constraints

- Must remain deterministic.
- Must not require source parsing.
- Must not infer business capabilities.

## Success Criteria

Given a valid Stacklit index, the export command writes one JSON artifact that:

- contains `schema_version`, `generator`, `generated_at`, and input fingerprint metadata
- exposes package/component identifiers derived deterministically from Stacklit modules
- exposes path-based membership suitable for joining with SCIP document paths
- represents missing optional relationships or entry points as empty arrays
- is produced without parsing source files or regenerating the Stacklit index
