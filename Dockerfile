FROM golang:1.24-alpine AS builder
WORKDIR /app

# Копируем зависимости и код
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gophermart ./cmd/gophermart

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/gophermart .

COPY migrations/ migrations/

EXPOSE 8080

CMD ["./gophermart"]