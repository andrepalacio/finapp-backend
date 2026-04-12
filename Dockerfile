# ── Stage: dev (hot reload with air) ─────────────────────────────────────────
FROM golang:1.22-alpine AS dev

RUN apk add --no-cache git && \
    go install github.com/air-verse/air@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

CMD ["air"]

# ── Stage: builder ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# ── Stage: runtime ────────────────────────────────────────────────────────────
FROM alpine:3.19 AS runtime

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080
ENTRYPOINT ["/app/server"]
