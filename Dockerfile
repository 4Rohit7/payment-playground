
# ---------- build ----------
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- run ----------
FROM postgres:16-alpine
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
COPY web /app/web
COPY docker-entrypoint-app.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
ENV PORT=8080
ENV PGDATA=/var/lib/postgresql/data
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=razorpay_playground
ENV DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/razorpay_playground?sslmode=disable
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]

# # ---------- build ----------
# FROM golang:1.24-alpine AS build
# WORKDIR /src
# COPY go.mod go.sum ./
# RUN go mod download
# COPY . .
# RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# # ---------- run ----------
# FROM alpine:3.20
# RUN apk add --no-cache ca-certificates tzdata \
#     && adduser -D -H -u 10001 app
# WORKDIR /app
# COPY --from=build /out/server /app/server
# COPY web /app/web
# USER app
# ENV PORT=8080
# EXPOSE 8080
# HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
#   CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1
# ENTRYPOINT ["/app/server"]