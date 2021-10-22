FROM --platform=${BUILDPLATFORM:-linux/amd64} teamserverless/license-check:0.3.9 as license-check
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.16 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ARG GIT_COMMIT
ARG VERSION

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOFLAGS=-mod=vendor
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

COPY --from=license-check /license-check /usr/bin/

RUN mkdir -p /go/src/github.com/openfaas-incubator/ingress-operator
WORKDIR /go/src/github.com/openfaas-incubator/ingress-operator

COPY . .

ARG OPTS
# RUN go mod download

RUN gofmt -l -d $(find . -type f -name '*.go' -not -path "./vendor/*")
RUN go test -mod=vendor -v ./...
RUN go build -mod=vendor -ldflags "-s -w \
  -X github.com/openfaas-incubator/ingress-operator/pkg/version.Release=${VERSION} \
  -X github.com/openfaas-incubator/ingress-operator/pkg/version.SHA=${GIT_COMMIT}" \
  -a -installsuffix cgo -o ingress-operator . && \
  addgroup --system app && \
  adduser --system --ingroup app app && \
  mkdir /scratch-tmp

# we can't add user in next stage because it's from scratch
# ca-certificates and tmp folder are also missing in scratch
# so we add all of it here and copy files in next stage

FROM scratch

LABEL org.opencontainers.image.source=https://github.com/openfaas/ingress-operator

COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=app:app /scratch-tmp /tmp/
COPY --from=builder /go/src/github.com/openfaas-incubator/ingress-operator/ingress-operator .

USER app

CMD ["./ingress-operator"]
