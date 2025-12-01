# Operator API Mirrors

This repository contains **standalone, version-pinned Go modules** extracted from upstream Kubernetes operators
(e.g. OpenTelemetry Operator, ECK).

Each operator version is published as a **reproducible, self-contained Go module** under:

```
mirrors/<operator>/<version>/
```

These modules include:

* the upstream **public API types** (e.g. `apis/*`, `pkg/apis/...`)
* only the **internal packages needed** to compile those types
* imports rewritten to reference this repository instead of upstream
* a minimal `go.mod` containing the upstream operator’s requirements
* optional dependency overrides applied at mirror-generation time

You can import these modules directly into your controllers without pulling the upstream operator’s full dependency graph.

---

## ✨ Why this repository exists

Upstream operators often package CRD types inside large monolithic repos. Importing those API types directly:

* drags in huge dependency trees
* forces your code to match the operator’s Kubernetes version
* introduces breaking changes upstream
* produces non-reproducible builds
* couples your code to internal implementation details

**This repository solves that.**

Each mirrored version is:

* small
* isolated
* stable
* pinned
* reproducible

Ideal for use in controllers or API clients that need operator CRD Go types without inheriting the operator itself.

---

## 📦 Repository structure

```
mirrors/
  ├── otel-operator/
  │   └── v0.140.0/
  │       ├── apis/
  │       ├── internal/
  │       ├── pkg/
  │       └── go.mod
  └── eck-operator/
      └── v3.2.0/
          ├── pkg/apis/elasticsearch/
          ├── internal/
          └── go.mod
```

Each version directory is a **complete Go module**.

You can import mirrored APIs like:

```go
import "github.com/sourcehawk/operator-api-mirrors/mirrors/otel-operator/v0.140.0/apis/v1beta1"
```

---

## 🧩 How these mirrors are generated

Mirrors in this repository are created using the companion project:

👉 **[https://github.com/sourcehawk/operator-api-mirrorer](https://github.com/sourcehawk/operator-api-mirrorer)**

The mirrorer tool:

1. Clones the upstream operator repo.
2. Copies API directories.
3. Detects internal package dependencies.
4. Rewrites import paths to point to this repository.
5. Builds a fresh `go.mod`.
6. Applies replace-overrides (e.g. to pin Kubernetes versions).
7. Runs `go mod tidy`.
8. Writes the result into this repository under `mirrors/<slug>/<version>/`.

This repository **does not contain the mirroring logic**—only the generated modules.

---

## 🛠 Consuming the Mirrored APIs

Each operator mirror is published as a Go module at:

```txt
github.com/sourcehawk/operator-api-mirrors/mirrors/<operator>
```

Git tags in this repository follow the pattern:

```txt
mirrors/<operator>/<operator-version>
```

Per [Go’s multi-module tagging rules](https://go.dev/doc/modules/managing-source#multiple-module-source), those tags 
allow you to `go get` a specific operator **version** using `@<operator-version>`.

### Fetching a mirrored API version

For example, to depend on the OpenTelemetry operator mirror at `v0.138.0`:

```bash
go get github.com/sourcehawk/operator-api-mirrors/mirrors/otel-operator@v0.138.0
```

This pins your project to that exact operator API version.

### Importing mirrored API types

Once added to your `go.mod`, you can import the mirrored API types like this:

```go
import (
    otelv1alpha1 "github.com/sourcehawk/operator-api-mirrors/mirrors/otel-operator/apis/v1alpha1"
)
```

Use the appropriate package path (`apis/...`, `pkg/apis/...`, etc.) for the operator you’re consuming.

### Guarantees

Every mirrored version is:

* **pinned** – tied to a specific operator version via `@<operator-version>`
* **reproducible** – generated in a controlled workflow from a known upstream commit
* **isolated** – no controllers, reconcilers, or extra operator internals
* **decoupled** – upstream changes won’t silently alter your dependency graph

This lets you use operator CRD Go types safely in your own controllers, tools, or clients without dragging in the operator itself.

---

## 🔄 Mirroring Workflow

The mirroring process for this repository works as follows:

1. **Renovate opens a PR** whenever it detects a new operator version in `operators.yaml` (in the mirrorer repo).
2. That PR triggers a **GitHub workflow** that:
    * runs the mirrorer tool,
    * regenerates all mirrors for the affected operator
    * commits any diffs back into the PR branch.
3. The PR is **manually reviewed and merged**.
4. After merging, another workflow runs that:

    * inspects the `currentVersion` for each operator in `operators.yaml`,
    * creates a **tag** for any version that does not already have one in the form
      `mirrors/<operator>/<operator-version>`.

Each generated mirror is therefore tied to a specific, reviewed version and published as a reproducible, versioned module.

---

### ❗ Why mirrors aren’t auto-merged

Leaving Renovate auto-merge enabled would remove control over version sequencing. This is a problem because:

* You might want to start mirroring an operator from an **earlier version** (e.g., begin at `1.0.0` even though `2.0.0` is the latest).
* If Renovate auto-merges the bump to `2.0.0` first, there is **no opportunity** to insert `1.0.0`, `1.1.0`, … in between.
* Mirrors in this repository are expected to be **monotonically increasing (semver)**, so skipped versions cannot be inserted later.

Manual merge ensures you retain full control over the version history and prevents inconsistent or broken mirror sequences.

## ❓ FAQ

### Does this replace upstream operators?

No — these modules contain only the **API types**, not controllers, reconcilers, or runtime logic.

### Why copy internal packages?

Some API code references upstream internals (e.g. helpers, version logic).
Only the required pieces are copied; all unknown or unused internals are excluded.

### Is licensing preserved?

Yes.
Upstream operators use Apache 2.0, and this repository preserves all upstream licenses.

---

## 🤝 Contributing

Contributions are welcome!

* Add support for more operators
* Update versions in `operators.yaml` (in the mirrorer repo)
* Submit a PR to publish new mirrors here

If you’re adding a new operator or need help generating its mirror, open an issue.

---

## 📝 License

Apache 2.0 — consistent with upstream operator sources.
