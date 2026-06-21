FROM golang:1.24 AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-server main.go

FROM alpine:3.18
COPY --from=builder /app/file-server /
ENTRYPOINT ["/file-server"]
