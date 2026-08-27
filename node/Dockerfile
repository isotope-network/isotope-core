FROM golang:1.25-alpine

WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем из main/main.go
RUN go build -o sbimain ./main

# Порт для HTTP
EXPOSE 8081

# Запускаем
CMD ["./sbimain"]