FROM golang:1.24 AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stomp cmd/stomp/main.go

FROM alpine:3.18
COPY --from=builder /app/stomp /
ENTRYPOINT ["/stomp"]
