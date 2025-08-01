FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download && go mod verify

COPY main.go ./
RUN GOOS=linux go build -ldflags="-s -w" -v -o /app/console-tool ./main.go
# Compress GoLang App binary. Lower size - x4
RUN apk add --no-cache --virtual .fetch-deps upx
RUN upx -1 /app/console-tool


FROM python:3.11-slim
RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir openai-whisper
COPY --from=builder /app/console-tool /app/console-tool

RUN addgroup --gid 501 usercode && \
    adduser --disabled-password \
    --gecos "" \
    --shell /usr/sbin/nologin \
    --ingroup usercode \
    --uid 501 \
    usercode
USER usercode

ENTRYPOINT ["/app/console-tool"]