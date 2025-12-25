FROM golang:1.24.11-alpine as builder

# We assume only git is needed for all dependencies.
# openssl is already built-in.
RUN apk add -U --no-cache git

WORKDIR /opt/wiilink/cfh

# Cache pulled dependencies if not updated.
COPY go.mod .
COPY go.sum .

# Copy necessary parts of the Mail-Go source into builder's source
COPY *.go ./
COPY assets assets
COPY templates templates
COPY middleware middleware

# Build to name "app".
RUN go build -o app .

EXPOSE 2001
# Wait until there's an actual MySQL connection we can use to start.
CMD ["./app"]