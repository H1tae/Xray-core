FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY xray /usr/local/bin/xray

RUN chmod +x /usr/local/bin/xray

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/xray"]
CMD ["run", "-config", "/app/server.json"]
