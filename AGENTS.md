# Notification Service - Agent Guide

A Kubernetes controller that watches Tekton PipelineRuns and sends notifications to AWS SNS when push pipeline runs complete successfully.

## Repository Structure

- `cmd/` - Binary entrypoint (main.go → manager)
- `internal/controller/` - Reconcile logic (not public API) and controller tests (envtest + Ginkgo)
- `pkg/notifier/` - Reusable notification abstraction (interface + SNS implementation)
- `config/` - Kustomize manifests (manager, rbac, prometheus, default overlay)
- `test/` - E2E tests and test utilities

## Tech Stack

- **Language**: Go 1.25.7
- **Framework**: controller-runtime (Kubebuilder patterns)
- **Watches**: Tekton PipelineRun (`tekton.dev/v1`)
- **Integration**: AWS SDK v2 (SNS)
- **Deployment**: Kustomize + Dockerfile (UBI multi-stage)

## Development Commands

### Setup
```bash
# Install dependencies
go mod download

# Generate manifests and code
make manifests generate
```

### Build & Run
```bash
# Build binary
make build

# Run locally (requires kubeconfig)
make run

# Build container image (uses podman by default; override with CONTAINER_TOOL=docker)
make docker-build

# Push image
make docker-push
```

### Testing
```bash
# Run tests (includes manifests, generate, fmt, vet, envtest setup, then go test)
make test

# Run e2e tests (requires Kind cluster, kubectl, Prometheus Operator, cert-manager)
make test-e2e

# Lint
make lint
```

### Deploy
```bash
# Generate manifests (outputs to config/crd/bases, requires kustomization.yaml setup)
make manifests

# Install CRDs (runs kustomize build config/crd)
# Note: config/crd directory and kustomization.yaml may need to be set up first
make install

# Deploy to cluster (namespace: notification-controller)
make deploy

# Undeploy
make undeploy
```

## Architecture Patterns

### Controller Pattern
- Uses **predicates** to filter events (only push PipelineRuns with specific labels)
- **Finalizer** (`konflux.ci/notification`) ensures cleanup on deletion
- **Annotation** (`konflux.ci/notified=true`) provides idempotency
- Event-driven reconciliation handles create, update, and deletion events

### Key Workflow
1. Push PipelineRun created → reconcile adds finalizer
2. Run completes successfully and not yet notified → extract results → call `Notifier.Notify()` → set annotation → remove finalizer
3. Run completes (failed or already notified) → remove finalizer without notification
4. Run deleted while still running → remove finalizer without notification
5. If any step fails (GetResults, Notify, AddAnnotation) → finalizer remains, reconcile retries later

### Separation of Concerns
- Reconciler in `internal/controller/` handles Kubernetes lifecycle
- `pkg/notifier/` abstracts notification mechanism (enables mocking, future backends)
- AWS credentials via env vars (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) or mounted secret (volume at `/.aws/`, expects file `/.aws/credentials`)

## Code Conventions

### RBAC
- Add `+kubebuilder:rbac` comments above reconciler for permission changes
- Run `make manifests` to regenerate RBAC manifests

### Testing
- Always add unit tests for new reconcile logic
- Use mock implementations of the `Notifier` interface for controller tests (see `MockNotifier` in suite_test.go)
- Use envtest for integration tests
- Run `make test` before committing

### Metrics
- Use Prometheus counters for observable events
- Existing metrics:
  - `notification_controller_notifications_total` - counts all notification attempts (including failures)
  - `notification_controller_notifications_failures_total` - counts only failed attempts
- Register new metrics in `internal/controller/metrics.go`

### Error Handling
- Return `ctrl.Result{}` with error to requeue
- Use finalizers carefully to avoid blocking deletion
- Log errors with context (namespace, name, operation)

## Pull Request Guidelines

### Before Submitting
1. Run `make test` and `make lint` - all tests must pass
2. Run `make manifests generate` if you changed APIs or RBAC
3. Add tests for new functionality
4. Update README.md if adding user-facing features

### Commit Messages
- Use conventional commits format: `type(scope): description`
- Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`
- Example: `feat(notifier): add Slack notification support`

### Review Expectations
- Code must pass CI (GitHub Actions + Tekton pipelines)
- Security: non-root containers, read-only root FS, dropped capabilities
- No secrets in code or manifests (use env vars or mounted secrets)

## Security Considerations

### AWS Credentials
- **Never commit** AWS keys or secrets
- Use mounted secrets (preferred) or env vars
- Secret volume should be mounted at `/.aws/` directory; SDK expects file at `/.aws/credentials`

### Dependencies
- Renovate manages dependency updates (see `renovate.json`)
- Review security advisories before merging dependency PRs
- Run `go mod tidy` after dependency changes

## Troubleshooting

### Controller Not Reconciling
- Check predicates match your PipelineRun labels (`pipelinesascode.tekton.dev/event-type=push`)
- Verify RBAC permissions for `pipelineruns` in API group `tekton.dev` (verbs: get, list, watch, update, patch)
- Check controller logs: `kubectl logs -n notification-controller deployment/notification-controller-controller-manager`

### SNS Publishing Failures
- Verify AWS credentials are mounted/set correctly
- Check `NOTIFICATION_TOPIC_ARN` and `NOTIFICATION_REGION` env vars
- Review `notification_controller_notifications_failures_total` metric

### Tests Failing
- Ensure `envtest` binaries are installed: `make envtest`
- Check Go version matches `go.mod` (1.25.7)
- For e2e tests, ensure cluster is accessible and namespace exists
- Note: E2E tests use namespace `notification-service-system`, while `make deploy` uses `notification-controller`

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/) - controller-runtime patterns
- [Tekton PipelineRun API](https://tekton.dev/docs/pipelines/pipelineruns/) - what we're watching
- [AWS SDK Go v2](https://aws.github.io/aws-sdk-go-v2/) - SNS client usage
