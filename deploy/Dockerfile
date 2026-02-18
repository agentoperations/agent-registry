FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/agent-registry ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/agent-registry /agent-registry

ENV PORT=8080
ENV DB_PATH=/data/registry.db

EXPOSE 8080
USER 65534:65534

ENTRYPOINT ["/agent-registry"]
