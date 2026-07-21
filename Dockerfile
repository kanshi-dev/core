FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o kanshi-core cmd/core/main.go


FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/kanshi-core .

EXPOSE 50051
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -q --spider http://127.0.0.1:8080/health || exit 1

CMD ["./kanshi-core"]
