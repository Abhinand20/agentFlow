# M18 - Build Manifest, Clean & Versioning

- Milestone: M18
- Version: v0.2+
- Status: **In progress**
- Depends on: M8 (CLI), M10 (Cursor binding)

## Goal

Record per-source build ownership of generated host artifacts so `af clean` can
remove files safely (scoped to one `.af` source) and `af versions` can inspect
build history and drift.

## Scope

### In scope

- Per-source manifest at `<target-root>/.agentflow/manifests/<slug>.json`
- `af build` writes/updates manifest (unless `--no-manifest`)
- `af build --prune` removes artifacts dropped since the previous build of the same source
- `af clean <file>` removes artifacts for one source; `--all` for every tracked source
- `af versions list|diff|status` for hash-only history and drift reporting
- Package `internal/manifest`

### Out of scope (deferred)

- Content rollback / `af rollback` (use git)
- Claude binding manifest path (lands with M7; schema is target-agnostic)
- Manifest import in `af import` (M17)

## Manifest layout

```
.cursor/.agentflow/manifests/<slug>.json
```

- `slug` = `<source-basename>-<8-hex-of-abs-path-sha256>` (e.g. `review-3f9a2c1b.json`)
- One file per `.af` source; enumeration via directory read (no index file)
- `history` capped at 20 builds (hash-only artifact records)

## CLI

```bash
af build examples/review.af --target cursor --out .
af build examples/review.af --target cursor --out . --prune
af clean examples/review.af --target cursor --out .
af clean --all --target cursor --out .
af versions list --target cursor --out .
af versions diff examples/review.af --target cursor --out .
af versions status examples/review.af --target cursor --out .
```

## Safety

- `clean` requires a source path or `--all` (never wipes everything by default)
- Drift guard (AF310): skip modified artifacts unless `--force`
- Cross-source guard (AF311): skip artifacts still owned by another source unless `--force`
- Overlap warning (AF312): warn at build when artifact paths collide across sources
- Invalid manifest skip (AF313): skip corrupt manifest files when loading directory
- Unreadable artifact skip (AF314): skip artifacts that cannot be read during clean
- Manifest slug keys on absolute source path (stable for a given file location; moving the repo creates a new slug)

## Packages & files

| Path | Role |
|------|------|
| `internal/manifest/` | Schema, build, load, diff, drift |
| `cmd/af/build.go` | Manifest write + `--prune` |
| `cmd/af/clean.go` | Source-scoped teardown |
| `cmd/af/versions.go` | History + drift commands |

## Acceptance

- [ ] `go test ./...` green
- [ ] `af build` writes per-source manifest JSON under `.cursor/.agentflow/manifests/`
- [ ] `af clean review.af` removes only that source's artifacts
- [ ] `af versions list` shows grouped sources; `diff`/`status` work per source
- [ ] Binding goldens unchanged (manifest layered in CLI, not `Emit`)
