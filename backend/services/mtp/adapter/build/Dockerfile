FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o adapter cmd/adapter/main.go

FROM alpine:3.18
COPY --from=builder /app/adapter /
ENTRYPOINT ["/adapter"]
