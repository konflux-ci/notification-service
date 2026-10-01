FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1790174511 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV GOTOOLCHAIN=local
WORKDIR /opt/app-root/src
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Prefetch (Hermeto/Cachi2) mounts modules under /cachi2; source the env when present
# so hermetic builds resolve deps offline. Konflux buildah also injects this before RUN.
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY internal/controller/ internal/controller/
COPY pkg/notifier/ pkg/notifier/

# Build from Hermeto-prefetched modules (no foreign Go binaries copied into the image).
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1790754119

COPY LICENSE /licenses
COPY --from=builder /opt/app-root/src/manager /
USER 65532:65532

LABEL name="Konflux Notification Service"
LABEL description="Konflux Notification Service"
LABEL com.redhat.component="konflux-notification-service-container"
LABEL io.k8s.description="Konflux Notification Service"
LABEL io.k8s.display-name="konflux-notification-service"
LABEL version="1.0"
LABEL release="1"
LABEL vendor="Red Hat, Inc."
LABEL distribution-scope="public"
LABEL url="https://github.com/konflux-ci/notification-service"

ENTRYPOINT ["/manager"]
