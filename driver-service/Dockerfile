FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o driver-service ./cmd

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/driver-service .

EXPOSE 8081

CMD ["./driver-service"]
