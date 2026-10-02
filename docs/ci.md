# CI / GitHub Actions

## Workflows

- **`test-yaml-consistency.yaml`**: PRs to `main` - runs `scripts/test-yaml-consistency.sh`.
- **`validate-no-offensive-lang.yml`**: PRs to `main` - language validation.
- **`release.yaml`**: On release published - builds multi-arch images, pushes to Quay, uploads manifest asset.

## Dependency management

- **Dependabot**: Watches Go modules and GitHub Actions (ignores Ginkgo/Gomega). Runs daily on
  `main` only, and opens PRs for all `gomod` updates, routine or security.
- **Renovate**: Runs on a schedule via `.github/workflows/renovate.yml` (GitHub App token; requires
  `RENOVATE_APP_ID`/`RENOVATE_APP_PRIVATE_KEY` repo secrets). Covers `main` and `release-v0.15`+
  branches, but only raises PRs for security fixes (including indirect deps), and runs
  `make vendor` and `make test` after each update. Excludes packages that typically need code
  changes (`k8s.io`, `kubevirt.io`, `sigs.k8s.io`, `openshift`, `knative.dev`, `tektoncd`,
  Ginkgo/Gomega). Also handles vulnerability/OSV alerts; ignores `vendor/`. Regular PR creation
  is limited to one per hour. Security vulnerability PRs bypass that hourly limit and may
  produce multiple open PRs in one run. On release branches, Renovate filters Go module updates
  for compatibility with each module's Go version, so fixes requiring a newer Go version may be
  skipped. Protobuf vulnerability fixes bypass this filter because Renovate's strict matching
  excludes modules declaring an older Go minor version, including the compatible v1.33.0 fix.
  Check the resulting PR's Go directive. A separate `pull_request_target` workflow checks changed
  `go.mod` files in same-repository Renovate PRs for `go` or `toolchain` directive changes. It
  runs the checker from the default branch and reads PR commits as data. When it finds a change,
  it posts a comment with the base and PR versions plus "Go version was bumped! Verify you can
  merge this PR!" Each Renovate push replaces the previous report comment; if the directive change
  is gone, the old comment is removed. Because Dependabot also covers security fixes on `main`,
  expect the occasional duplicate PR there; release branches only get updates from Renovate.

---
<- Back to [AGENTS.md](../AGENTS.md) | [Documentation Index](../AGENTS.md#documentation)
