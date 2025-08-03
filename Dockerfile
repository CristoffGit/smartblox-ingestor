FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o smartblox-ingestor .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/smartblox-ingestor .
CMD ["./smartblox-ingestor"]