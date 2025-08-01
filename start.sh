#!/bin/bash

# Настройка переменных окружения
export GITCALL_PORT=8080
export UPLOAD_URL="https://mw-api.simulator.company/v/1.0/upload/1234"
export ACCESS_TOKEN="1234"

# Запуск сервера
go run main.go