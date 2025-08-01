FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download && go mod verify

COPY main.go ./
RUN GOOS=linux go build -ldflags="-s -w" -v -o /app/console-tool ./main.go
# Compress GoLang App binary. Lower size - x4
RUN apk add --no-cache --virtual .fetch-deps upx
RUN upx -1 /app/console-tool


FROM ubuntu:22.04
COPY --from=builder /app/console-tool /app/console-tool

# Install dependencies and make it look like a regular desktop
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    ffmpeg \
    curl \
    wget \
    ca-certificates \
    xvfb \
    firefox \
    chromium-browser \
    && rm -rf /var/lib/apt/lists/*

# Install latest yt-dlp
RUN pip3 install --no-cache-dir --upgrade yt-dlp

# Create user directories that browsers expect
RUN groupadd --gid 501 usercode && \
    useradd --uid 501 --gid usercode --shell /bin/bash --create-home usercode

# Set up browser directories
RUN mkdir -p /home/usercode/.mozilla/firefox \
    /home/usercode/.config/google-chrome \
    /home/usercode/.cache \
    && chown -R usercode:usercode /home/usercode

USER usercode
WORKDIR /home/usercode

# Set environment variables to make it look more like a desktop
ENV DISPLAY=:99
ENV HOME=/home/usercode

ENTRYPOINT ["/app/console-tool"]