---
name: debug-running-instance
description: >-
  Debug a running instance of the notification-service controller. Use when
  investigating why notifications are not being sent, PipelineRuns are stuck
  with finalizers, or the controller pod is unhealthy.
---

# Debugging a Running Instance

## Controller Pod

The controller deploys to namespace `notification-controller` with label
`control-plane=controller-manager`.

```bash
# Find the pod
kubectl get pods -n notification-controller -l control-plane=controller-manager

# Stream logs
kubectl logs -n notification-controller -l control-plane=controller-manager -f

# Check pod events
kubectl describe pod -n notification-controller -l control-plane=controller-manager
```

## Health and Readiness

The controller exposes health probes on port 8081:

```bash
# From inside the cluster or via port-forward
kubectl port-forward -n notification-controller deploy/notification-controller-controller-manager 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

## Prometheus Metrics

Two custom metrics are exposed:

| Metric | Description |
|--------|-------------|
| `notification_controller_notifications_total` | Total notification attempts |
| `notification_controller_notifications_failures_total` | Failed notifications |

Access via the metrics endpoint (HTTPS by default, port 8443):

```bash
kubectl port-forward -n notification-controller deploy/notification-controller-controller-manager 8443:8443
```

A `ServiceMonitor` is deployed if prometheus-operator CRDs are available.

## AWS / SNS Issues

### Credentials

The controller mounts credentials from secret `aws-sns-secret` at `/.aws/credentials`:

```bash
# Verify the secret exists
kubectl get secret aws-sns-secret -n notification-controller

# Check if the volume is mounted
kubectl describe pod -n notification-controller -l control-plane=controller-manager | grep -A3 "vol-secret"
```

Expected credentials file format:

```
[default]
aws_access_key_id=<KEY>
aws_secret_access_key=<SECRET>
```

If the secret is missing, the pod stays in `ContainerCreating`.

### Environment Variables

```bash
kubectl get deploy -n notification-controller notification-controller-controller-manager \
  -o jsonpath='{.spec.template.spec.containers[0].env}' | jq .
```

| Variable | Required | What to check |
|----------|----------|---------------|
| `NOTIFICATION_TOPIC_ARN` | Yes | Must be a valid SNS ARN |
| `NOTIFICATION_REGION` | Yes | Must match the topic's region |
| `NOTIFICATION_FILTER_LABELS` | No | Comma-separated `key=value` pairs |
| `NOTIFICATION_FILTER_ANNOTATIONS` | No | Comma-separated annotation keys |

If `NOTIFICATION_TOPIC_ARN` or `NOTIFICATION_REGION` are unset, the controller starts
but every notification attempt fails.

## PipelineRun State

### Finalizer stuck

If a PipelineRun cannot be deleted, check for the controller's finalizer:

```bash
kubectl get pipelinerun <name> -o jsonpath='{.metadata.finalizers}'
```

Expected finalizer: `konflux.ci/notification`

The controller removes the finalizer after:
- Successful notification (run succeeded)
- Run failed (no notification sent, finalizer removed)
- Run deleted while still running (finalizer removed to allow GC)

**Most common cause of stuck finalizers**: `Notify()` failure. When a PipelineRun
succeeds but notification fails (bad credentials, missing env vars, SNS error), the
reconciler returns early and never reaches the finalizer removal. The run requeues
indefinitely. Check the logs for "Failed to Notify" and verify SNS configuration.

Other causes:
- Controller is down or crashlooping
- RBAC issue (controller can't patch PipelineRuns)
- Namespace mismatch

To manually remove a stuck finalizer:

```bash
kubectl patch pipelinerun <name> -p '{"metadata":{"finalizers":null}}' --type=merge
```

### Notification already sent

Check if the annotation is set:

```bash
kubectl get pipelinerun <name> -o jsonpath='{.metadata.annotations.konflux\.ci/notified}'
```

If value is `true`, the notification was already sent and the controller won't re-notify.

### PipelineRun not being processed

The controller only processes PipelineRuns matching its filter. Default filter:
`pipelinesascode.tekton.dev/event-type=push`.

```bash
# Check if the PipelineRun has the expected label
kubectl get pipelinerun <name> -o jsonpath='{.metadata.labels.pipelinesascode\.tekton\.dev/event-type}'
```

If using custom filters, check the `NOTIFICATION_FILTER_LABELS` and
`NOTIFICATION_FILTER_ANNOTATIONS` env vars on the deployment.

## Common Log Messages

| Log message | Meaning |
|-------------|---------|
| "Trying to add finalizer" | PipelineRun matched filter, controller is tracking it |
| "Finalizer was added" | Successfully patched the PipelineRun |
| "Pipelinerun ended" | Run completed (check `ended_successfully` field) |
| "SNS Notified" | Notification sent successfully |
| "Failed to Notify" | SNS publish failed (check AWS creds/ARN/region) |
| "Failed to get pipelineRun" | PipelineRun disappeared or RBAC issue |
| "Conflict removing finalizer" | Concurrent update, will retry automatically |

## Leader Election

If running multiple replicas, only the leader reconciles. Check leader election:

```bash
kubectl get lease -n notification-controller
```

The lease ID is `75765374.konflux.ci`. If the leader pod is stuck, delete it to trigger
re-election.
