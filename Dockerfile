FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/wechat-clone ./cmd/main.go

FROM debian:bookworm-slim AS runtime

WORKDIR /app

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates tzdata \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/wechat-clone /app/wechat-clone
COPY migration /app/migration

EXPOSE 35000

ENTRYPOINT ["/app/wechat-clone", "-path", "/app/migration"]