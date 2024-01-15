FROM golang:1.19 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/github.com/traas-stack/altershield-operator

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . /go/src/github.com/traas-stack/altershield-operator

RUN cd /go/src/github.com/traas-stack/altershield-operator/certs && \
    sh generate-tls-certificates.sh

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
#RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go
RUN GO111MODULE=on CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot
FROM alpine:3.16 AS final
RUN apk update && apk add curl
WORKDIR /
COPY --from=builder /go/src/github.com/traas-stack/altershield-operator/manager .
COPY --from=builder /go/src/github.com/traas-stack/altershield-operator/certs/tls.crt /tmp/k8s-webhook-server/serving-certs/tls.crt
COPY --from=builder /go/src/github.com/traas-stack/altershield-operator/certs/tls.key /tmp/k8s-webhook-server/serving-certs/tls.key
USER 65532:65532
#USER root

ENTRYPOINT ["/manager"]
