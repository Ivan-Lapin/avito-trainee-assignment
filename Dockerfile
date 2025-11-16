# build
FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /out/pr-reviewer ./app/cmd/server

RUN test -f /out/pr-reviewer && (ldd /out/pr-reviewer || true)

# runtime: статический образ
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/pr-reviewer /pr-reviewer
EXPOSE 8095
USER nonroot:nonroot
ENTRYPOINT ["/pr-reviewer"]
