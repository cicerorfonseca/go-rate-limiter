# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# layer-caching layer across rebuilds as long
# as go.mod itself hasn't changed
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/rate-limiter ./cmd/server

# ---- run stage ----
FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /out/rate-limiter /usr/local/bin/rate-limiter
USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/rate-limiter"]