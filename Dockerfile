# ---- Build Stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY --link go.mod go.sum ./
RUN go mod download

COPY --link . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dashboard-server .

# ---- Runtime Stage ----
FROM alpine:3.21

WORKDIR /app

COPY --link static/ ./static/

COPY --link --from=builder /app/dashboard-server /usr/local/bin/dashboard-server

EXPOSE 9099

ENV SERVER_PORT="9099"

CMD ["dashboard-server"]