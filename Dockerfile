# Build image
FROM golang:1.14-alpine3.11 AS builder

ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial bzr

RUN mkdir -p /workspace
WORKDIR /workspace

ARG GOPROXY

COPY go.* ./
RUN go mod download

ARG BUILD_TARGET

COPY Makefile *.mk ./

RUN if [[ "${BUILD_TARGET}" == "debug" ]]; then make build-debug-deps; else make build-release-deps; fi

COPY . .

RUN set -xe && \
    if [[ "${BUILD_TARGET}" == "debug" ]]; then \
        cd /tmp; GOBIN=/workspace/build/debug go get github.com/go-delve/delve/cmd/dlv; cd -; \
        make build-debug; \
        mv build/debug /build; \
    else \
        make build-release; \
        mv build/release /build; \
    fi


# Final image
FROM alpine:3.11

RUN apk add --update --no-cache ca-certificates tzdata bash curl

SHELL ["/bin/bash", "-c"]

ARG BUILD_TARGET

RUN if [[ "${BUILD_TARGET}" == "debug" ]]; then apk add --update --no-cache libc6-compat; fi

COPY --from=builder /build/* /usr/local/bin/

EXPOSE 8080
CMD ["binomo-bot"]