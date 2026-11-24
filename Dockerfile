# ---- Build Stage ----
FROM golang:1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

WORKDIR /app

COPY --link go.mod go.sum ./
RUN go mod download

COPY --link . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dashboard-server .

# ---- Runtime Stage ----
FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

WORKDIR /app

COPY --link static/ ./static/

COPY --link --from=builder /app/dashboard-server /app/dashboard-server

EXPOSE 9099

ENV SERVER_PORT="9099"

CMD ["/app/dashboard-server"]