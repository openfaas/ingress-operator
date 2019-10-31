ARG arch=amd64
FROM golang:1.11 AS builder

RUN mkdir -p /go/src/github.com/openfaas-incubator/ingress-operator/

WORKDIR /go/src/github.com/openfaas-incubator/ingress-operator

COPY . .

ARG go_opts
RUN gofmt -l -d $(find . -type f -name '*.go' -not -path "./vendor/*") && \
  VERSION=$(git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///') && \
  GIT_COMMIT=$(git rev-list -1 HEAD) && \
  env $go_opts CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w \
  -X github.com/openfaas-incubator/ingress-operator/pkg/version.Release=${VERSION} \
  -X github.com/openfaas-incubator/ingress-operator/pkg/version.SHA=${GIT_COMMIT}" \
  -a -installsuffix cgo -o ingress-operator .

FROM alpine:3.10 AS staging

RUN addgroup -S app \
    && adduser -S -g app app

WORKDIR /home/app

COPY --from=0 /go/src/github.com/openfaas-incubator/ingress-operator/ingress-operator .

RUN chown -R app:app ./

FROM $arch/alpine:3.10

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=staging /etc/passwd /etc/group /etc/
COPY --from=staging /home/app/ /home/app/

WORKDIR /home/app
USER app

ENTRYPOINT ["./ingress-operator"]
CMD ["-logtostderr"]
