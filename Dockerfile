# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src

# Install build deps (optional, keeps cache saner)
RUN apk add --no-cache ca-certificates

# Go module download first (better caching)
COPY go.mod ./
# COPY go.sum ./
RUN go mod download

# Bring in the code
COPY . .

# Build static-ish binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app ./cmd/server

# --- runtime stage ---
FROM alpine:3.19
WORKDIR /app
COPY --from=build /bin/app /app/app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run as non-root
RUN adduser -D -H appuser
USER appuser

# The app listens on 8080
EXPOSE 8080
ENTRYPOINT ["/app/app"]
