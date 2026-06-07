---
name: run-tests
description: >-
  Run unit tests, integration tests, and e2e tests for the notification-service
  controller. Use when the user asks to run tests, debug test failures, or wants
  to know how to execute a specific test.
---

# Running Tests

## Unit and Integration Tests

```bash
make test
```

This target chains `manifests`, `generate`, `fmt`, `vet`, and `envtest` before running:

```bash
KUBEBUILDER_ASSETS="$(setup-envtest use 1.30.x ...)" go test $(go list ./... | grep -v /e2e) -coverprofile cover.out
```

### What happens under the hood

1. `setup-envtest` downloads kubebuilder binaries (kube-apiserver, etcd) to `bin/k8s/1.30.0-<os>-<arch>`
2. Tests in `internal/controller/` use **envtest** to spin up a real kube-apiserver + etcd
3. Tekton PipelineRun CRDs are loaded from `tektoncd/pipeline` module's config directory (resolved via `operator-toolkit/test.GetRelativeDependencyPath`)
4. The controller manager starts in a goroutine per test (`BeforeEach`/`AfterEach` lifecycle)

### Running a single controller test

```bash
# Focus on a specific Ginkgo test by description
go test ./internal/controller/ -run TestControllers -v -- -ginkgo.focus "should add finalizer"
```

### Running notifier unit tests

The `pkg/notifier/` package uses standard Go `testing` (not Ginkgo):

```bash
go test ./pkg/notifier/ -v
```

### Common failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| `failed to start control plane` | Missing kubebuilder binaries | Run `make envtest` or delete `bin/k8s/` and re-run `make test` |
| GCS 401 downloading envtest assets | Exact version not found | Already mitigated by using wildcard `1.30.x` in `ENVTEST_K8S_VERSION` |
| `CRDDirectoryPaths` not found | `GOPATH/pkg/mod` missing tektoncd/pipeline | Run `go mod download` |
| Reflector watch errors | Cache race | The test suite disables cache for PipelineRun objects to avoid this |

## E2E Tests

```bash
make test-e2e
```

Runs `go test ./test/e2e/ -v -ginkgo.v` against whatever cluster your kubeconfig points to.

**Important**: The e2e suite is Kubebuilder scaffold and is not run in CI. The `make install`
step inside the test references `config/crd/` which does not exist in this repo (this project
doesn't own a CRD). The e2e tests may need modification to work end-to-end.

For manual deployment testing on a real cluster, see the `local-dev-setup` skill.

## Lint

```bash
make lint
```

Runs `golangci-lint` with a 5-minute timeout. The `ginkgolinter` rule is enabled and will fail on non-idiomatic Ginkgo/Gomega assertions.

## Coverage

After `make test`, coverage is written to `cover.out`. View it with:

```bash
go tool cover -html=cover.out
```
