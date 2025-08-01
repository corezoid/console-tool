#!/bin/bash

#curl -X POST http://localhost:8080 \
#  -H "Content-Type: application/json" \
#  -d '{
#    "jsonrpc": "2.0",
#    "id": "test123",
#    "method": "execute",
#    "params": {
#      "command_name": "echo",
#      "command_args": ["Hello, World!"]
#    }
#  }'
#
#echo -e "\n\n--- Создание файла ---"


curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "test456", 
    "method": "execute",
    "params": {
      "command_name": "yt-dlp",
      "command_args": ["--extract-audio","--audio-format", "mp3", "https://www.youtube.com/watch?v=1234"]
    }
  }'