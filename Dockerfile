FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o estatelink-api ./cmd/api

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/estatelink-api .

EXPOSE 8080

CMD ["./estatelink-api"]