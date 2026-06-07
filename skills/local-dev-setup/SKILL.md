---
name: local-dev-setup
description: >-
  Set up a local development environment to run and test the notification-service
  controller against a real Kubernetes cluster. Use when the user wants to deploy
  the controller locally, manually test PipelineRun reconciliation, or iterate
  on changes.
---

# Local Development Setup

## Prerequisites

A Kubernetes cluster (Kind, OpenShift, etc.) with:
- Tekton Pipelines installed (`kubectl get crd pipelineruns.tekton.dev` to verify)
- kubectl context configured (`~/.kube/config` or `$KUBECONFIG`)

## Option A: Run Controller on Host

The fastest way to iterate. The controller runs on your machine, talking to whatever
cluster your kubeconfig points to.

```bash
make run
```

`make run` chains `manifests generate fmt vet` before `go run ./cmd/main.go`.
The controller starts and watches PipelineRuns immediately without env vars.

**Important**: Without valid SNS config (`NOTIFICATION_TOPIC_ARN`, `NOTIFICATION_REGION`,
AWS credentials), the controller adds finalizers to matching PipelineRuns but when a run
succeeds, `Notify()` fails and the reconciler returns early -- the finalizer is never
removed. This leaves successful PipelineRuns stuck (requeuing indefinitely). Failed
PipelineRuns are unaffected (finalizer is removed without attempting notification).

For local development:

```bash
NOTIFICATION_TOPIC_ARN=arn:aws:sns:us-east-1:000000000000:test \
NOTIFICATION_REGION=us-east-1 \
AWS_ACCESS_KEY_ID=FAKE \
AWS_SECRET_ACCESS_KEY=FAKE \
  make run
```

Even with fake credentials, `Notify()` will still fail and successful PipelineRuns will
have stuck finalizers. To manually unstick:
`kubectl patch pipelinerun <name> -p '{"metadata":{"finalizers":null}}' --type=merge`

## Option B: Deploy as Container

For testing the full deployment (image build, RBAC, volume mounts):

```bash
make docker-build IMG=example.com/notification-service:v0.0.1
make deploy IMG=example.com/notification-service:v0.0.1
```

If deploying to Kind, load the image first:

```bash
kind load docker-image example.com/notification-service:v0.0.1 --name <cluster-name>
```

The deployment requires an `aws-sns-secret` secret or the pod stays in `ContainerCreating`:

```bash
kubectl create secret generic aws-sns-secret \
  --from-literal=credentials='[default]
aws_access_key_id=FAKE
aws_secret_access_key=FAKE' \
  -n notification-controller
```

`make deploy` will fail (non-zero exit) if prometheus-operator CRDs are missing, because
the kustomize output includes a `ServiceMonitor` resource. The controller Deployment is
still created (it's applied before the ServiceMonitor), but the command reports failure.

Workarounds:
- Install prometheus-operator CRDs first:
  `kubectl apply --filename https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.72.0/bundle.yaml`
- Or remove `../prometheus` from `config/default/kustomization.yaml` for local testing.

### Undeploy

```bash
make undeploy
```

## Testing a PipelineRun Notification

Once the controller is running (Option A or B), create a test PipelineRun:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: test-notification
  labels:
    pipelinesascode.tekton.dev/event-type: push
spec:
  pipelineSpec:
    tasks:
      - name: hello
        taskSpec:
          steps:
            - name: say-hello
              image: alpine
              script: echo hello
EOF
```

Watch the controller logs for:
- "Trying to add finalizer" -- on creation
- "Pipelinerun ended" -- on completion
- "SNS Notified" or "Failed to Notify" -- notification attempt

## Key Notes


- The controller deploys to namespace `notification-controller`.
- Default filter: only PipelineRuns with label `pipelinesascode.tekton.dev/event-type=push`.
  Override with `NOTIFICATION_FILTER_LABELS` env var.
- Container builds default to `podman` (override: `CONTAINER_TOOL=docker`).
- Image tag override: `TAG=my-tag make docker-build` or `IMG=quay.io/user/ns:my-tag make docker-build`.
