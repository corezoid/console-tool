FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache curl
RUN curl -o main.go https://raw.githubusercontent.com/corezoid/console-tool/main/main.go
RUN go mod init console-tool && go mod tidy && GOOS=linux go build -ldflags="-s -w" -v -o /app/console-tool ./main.go
# Compress GoLang App binary. Lower size - x4
RUN apk add --no-cache --virtual .fetch-deps upx
RUN upx -1 /app/console-tool


FROM python:3.11-slim
COPY --from=builder /app/console-tool /app/console-tool

# TODO: Install required packages

RUN addgroup --gid 501 usercode && \
    adduser --disabled-password \
    --gecos "" \
    --shell /usr/sbin/nologin \
    --ingroup usercode \
    --uid 501 \
    usercode
USER usercode

ENTRYPOINT ["/app/console-tool"]