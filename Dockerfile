FROM golang:1.25-bookworm AS builder

WORKDIR /src/xray

COPY go.mod go.sum /src/xray/

RUN go mod download

COPY . /src/xray

RUN CGO_ENABLED=0 go build \
    -o /out/xray \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w -buildid=" \
    -v ./main


FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/xray /usr/local/bin/xray

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/xray"]
CMD ["run", "-config", "/app/server.json"]
