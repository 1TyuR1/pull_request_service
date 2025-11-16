FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/pr-reviewer-service ./cmd/app

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/pr-reviewer-service .

EXPOSE 8080
CMD ["./pr-reviewer-service"]
