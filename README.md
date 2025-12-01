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

## 🔄 Updating mirrors

To update mirrors:

* Use **Renovate** or GitHub Actions in this repository (optional)
* Run the mirrorer tool in the mirrorer repository
* Commit the generated mirrors here

If you'd like an example GitHub Actions workflow for automatically updating mirrors, just ask!

---

## 🛠 Consuming the mirrored APIs

You can import any mirrored API version directly:

```go
import "github.com/sourcehawk/operator-api-mirrors/mirrors/eck-operator/v3.2.0/pkg/apis/elasticsearch/v1"
```

Every version is:

* pinned
* reproducible
* isolated
* free from upstream implementation packages

This ensures your project does not accidentally upgrade operator dependencies or break when upstream changes.

---

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
