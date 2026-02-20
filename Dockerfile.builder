FROM docker.io/golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum .
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /builder ./cmd/builder

FROM docker.io/alpine:latest
RUN apk add --no-cache make
COPY --from=builder /builder /builder
WORKDIR /repos
ENTRYPOINT ["/builder"]
