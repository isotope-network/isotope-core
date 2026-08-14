FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sbimain .   

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sbimain .
COPY genesis/ethics_hash.txt ./genesis/
COPY index.html /root/index.html
COPY layers.html /root/layers.html
EXPOSE 9000 9001
CMD ["./sbimain"]