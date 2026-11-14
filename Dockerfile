FROM golang:1.26.1-alpine AS builder

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/aibuddy \
    ./cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder --chown=app:app /out/aibuddy ./aibuddy

USER app
EXPOSE 8080

ENTRYPOINT ["./aibuddy"]
