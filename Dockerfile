FROM golang:1.24.11-alpine AS builder

# We assume only git is needed for all dependencies.
# openssl is already built-in.
RUN apk add -U --no-cache git

WORKDIR /app

# Cache pulled dependencies if not updated.
COPY go.mod .
COPY go.sum .

# Copy necessary parts of the Mail-Go source into builder's source
COPY *.go ./
COPY middleware middleware

# Build to name "app".
RUN go build -o app .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/app .
COPY templates templates
COPY assets assets

EXPOSE 2001
# Wait until there's an actual MySQL connection we can use to start.
CMD ["./app"]
