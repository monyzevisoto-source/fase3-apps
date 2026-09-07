# Build
FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app /app

CMD ["/app"]