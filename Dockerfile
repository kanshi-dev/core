FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o kanshi-core cmd/core/main.go


FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/kanshi-core .

EXPOSE 50051
EXPOSE 8080

CMD ["./kanshi-core"]