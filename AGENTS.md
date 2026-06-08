# Agent Guidelines for notification-service

## Project Overview

Kubernetes controller (controller-runtime / Kubebuilder v4) that watches Tekton
`PipelineRun` objects filtered by configurable labels/annotations. When a matching
run succeeds, it extracts results, adds metadata (name, namespace, application),
and publishes the JSON payload to AWS SNS. A finalizer (`konflux.ci/notification`)
ensures completion is observed; an annotation (`konflux.ci/notified=true`) prevents
duplicate notifications.

## Project Structure

```
cmd/main.go                 - Entrypoint: wires manager, SNS notifier, reconciler
internal/controller/
  notificationservice_controller.go - Reconcile loop and SetupWithManager
  predicates.go             - Event filter predicates (create, update, delete)
  pipelinerun_helper.go     - Finalizer, annotation, result extraction, filtering logic
  metrics.go                - Prometheus counter definitions and registration
  suite_test.go             - Ginkgo test suite setup: envtest, mock notifier, mock client
  notificationservice_controller_test.go - Controller integration tests
  pipelinerun_helper_test.go - Unit tests for helper functions
pkg/notifier/
  notifier.go               - Notifier/Publisher interfaces and SNSNotifier implementation
  notifier_test.go          - Notifier unit tests (standard Go testing, not Ginkgo)
config/
  default/                  - Kustomize overlay combining manager + RBAC + prometheus
  manager/                  - Controller Deployment manifest
  rbac/                     - ClusterRole, ClusterRoleBinding, ServiceAccount
  prometheus/               - ServiceMonitor for metrics scraping
test/e2e/                   - Ginkgo Ordered e2e tests (require Kind cluster)
hack/                       - License header boilerplate for code generation
.tekton/                    - Konflux CI pipeline definitions (PipelineRun templates)
.github/workflows/          - GitHub Actions: lint, unit tests, fullsend automation
```

## Reconciliation Lifecycle

The controller watches `PipelineRun` objects via four composed predicates
(`SetupWithManager` in `notificationservice_controller.go`):

1. `PipelineRunCreatedPredicate` -- triggers on create for matching PipelineRuns
2. `PipelineRunEndedFinalizerPredicate` -- triggers on update when run ends and has finalizer
3. `PipelineRunEndedNoAnnotationPredicate` -- triggers on update when run ends without notification annotation
4. `PipelineRunDeletingPredicate` -- triggers when a running PipelineRun is deleted

The reconcile flow:

```
PipelineRun created (matching filter)
  └─ Add finalizer `konflux.ci/notification`
PipelineRun still running but deleted
  └─ Remove finalizer (allow garbage collection)
PipelineRun ended successfully (no annotation yet)
  └─ Extract results via GetResultsFromPipelineRun
  └─ Call Notifier.Notify (publishes to SNS)
  └─ Increment notification_controller_notifications_total
  └─ On notify error: increment notification_controller_notifications_failures_total, requeue
  └─ Add annotation `konflux.ci/notified=true`
  └─ Remove finalizer
PipelineRun ended with failure
  └─ Remove finalizer (no notification sent)
```

Results payload includes `PIPELINERUN_NAME`, `NAMESPACE`, `APPLICATION` (from
the `appstudio.openshift.io/application` label) plus all PipelineRun status results,
serialized as JSON.

## Key Interfaces and Abstractions

```go
// pkg/notifier/notifier.go
type Notifier interface {
    Notify(context.Context, string) error
}

type Publisher interface {
    Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type ClientRefresher func() (Publisher, error)
```

- `Notifier` is the main abstraction; the reconciler depends on it (allows mocking in tests)
- `Publisher` wraps the AWS SNS `Publish` call (allows mocking the AWS SDK)
- `ClientRefresher` is called on every `Notify` to get a fresh SNS client
- `SNSNotifier` is the production implementation wiring all three together

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `NOTIFICATION_TOPIC_ARN` | Yes | SNS topic ARN for publishing messages |
| `NOTIFICATION_REGION` | Yes | AWS region for SNS client |
| `NOTIFICATION_FILTER_LABELS` | No | Comma-separated `key=value` label filters |
| `NOTIFICATION_FILTER_ANNOTATIONS` | No | Comma-separated annotation-key presence filters |

- Default filter (when both filter vars are unset): `pipelinesascode.tekton.dev/event-type=push`
- AWS credentials: mounted secret at `/.aws/credentials` (preferred), or
  `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` environment variables

## Build and Test

```
make test          # unit/integration tests (envtest + Ginkgo/Gomega), generates cover.out
make lint          # golangci-lint (ginkgolinter enabled, 5m timeout)
make build         # builds bin/manager
make test-e2e      # e2e tests against Kind cluster
make docker-build  # container build (CONTAINER_TOOL defaults to podman)
make docker-push   # push container image
make manifests     # regenerate RBAC/CRD manifests via controller-gen
make generate      # regenerate DeepCopy methods
make fmt           # go fmt
make vet           # go vet
```

## Test Patterns

### Controller tests (`internal/controller/`)

- Framework: **Ginkgo v2 + Gomega** with dot imports (`. "github.com/onsi/ginkgo/v2"`)
- Infrastructure: **envtest** provides a real kube-apiserver + etcd; Tekton CRDs loaded
  from the `tektoncd/pipeline` module's config directory via `operator-toolkit/test`
- Mocking: `MockNotifier` implements `Notifier` interface with a call counter and
  configurable error; `clientMock` wraps `client.Client` to simulate patch failures
- Test setup: `BeforeEach` bootstraps a fresh envtest environment and manager per test;
  `AfterEach` tears it down. The manager is started in a goroutine
- Metric assertions: use `prometheus/client_golang/prometheus/testutil.ToFloat64()` to
  read counter values before and after reconciliation
- Pattern for async assertions: `Eventually(func() bool { ... }, timeout, interval).Should(BeTrue())`
  with `timeout = 10s` and `interval = 250ms`
- PipelineRun fixtures: create with `Status.Conditions` set to `Unknown` (running),
  then update via `k8sClient.Status().Update()` to `True` (success) or `False` (failure)

### Notifier tests (`pkg/notifier/`)

- Framework: **standard Go `testing` package** (not Ginkgo)
- Mocking: `MockSNSPublisher` implements `Publisher` interface; mock `ClientRefresher`
  functions for success/failure scenarios
- Tests cover: successful notify, failed client refresh, failed publish

## Conventions

- All controller/integration tests use Ginkgo v2 + Gomega; the `ginkgolinter` lint rule is enforced
- Notifier package uses standard Go `testing` -- keep this consistent
- controller-runtime patterns: predicates filter events, finalizers track lifecycle
- Tekton PipelineRun v1 API (`github.com/tektoncd/pipeline/pkg/apis/pipeline/v1`)
- Apache 2.0 license headers on all Go source files (template: `hack/boilerplate.go.txt`)
- Prometheus metrics are defined and registered centrally in `internal/controller/metrics.go`
- Container builds use `podman` by default; override with `CONTAINER_TOOL=docker`
- Konflux CI pipelines in `.tekton/` handle container image builds; GitHub Actions handle lint and tests

## Kustomize Layout

```
config/
  default/     - Main overlay: combines manager + RBAC + prometheus
  manager/     - Deployment for the controller (image: controller:latest)
  rbac/        - ClusterRole (tekton.dev PipelineRuns CRUD), ServiceAccount, bindings
  prometheus/  - ServiceMonitor scraping the /metrics endpoint
```

Note: there is no `config/crd/` directory. The controller does not own a CRD -- it watches
Tekton's `PipelineRun` resource. The `make install`/`make uninstall` Makefile targets reference
`config/crd` from the Kubebuilder scaffold but will fail since the directory does not exist.

## Dependency Notes

- `go.mod` contains a `replace` directive pinning `github.com/google/cel-go` to `v0.28.1`.
  Preserve this when updating dependencies
- `github.com/konflux-ci/operator-toolkit` provides metadata helpers (`HasLabelWithValue`,
  `HasAnnotation`, `SetAnnotation`) and test utilities (`GetRelativeDependencyPath`)
- `knative.dev/pkg/apis` provides `ConditionSucceeded` for checking PipelineRun status

## CI and Pipelines

- **GitHub Actions** (`.github/workflows/`):
  - `go-lint.yaml` -- `make lint` + AGENTS.md line count validation
  - `go-test.yaml` -- `make test` + Codecov upload (OIDC)
  - `fullsend.yaml` -- Agent automation dispatch (review/fix/triage)
- **Tekton / Konflux** (`.tekton/`):
  - `build-pipeline.yaml` -- Multi-arch trusted-artifact pipeline with Cachi2 prefetch (gomod)
  - `notification-service-pull-request.yaml` -- PaC PipelineRun for PRs
  - `notification-service-push.yaml` -- PaC PipelineRun for pushes to main

## AI Skills

Skills live in `skills/` and are symlinked to `.cursor/skills` and `.claude/skills` for IDE discovery.

| Skill | Purpose |
|-------|---------|
| run-tests | How to run unit and e2e tests |
| local-dev-setup | Running the controller locally or deploying to a cluster |
| debug-running-instance | Debugging a deployed controller instance |
