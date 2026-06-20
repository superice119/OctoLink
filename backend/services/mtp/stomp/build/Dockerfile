FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stomp cmd/stomp/main.go

FROM alpine:3.18
COPY --from=builder /app/stomp /
ENTRYPOINT ["/stomp"]
