FROM golang:1.24 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make release

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/build/hatchery /usr/local/bin/hatchery

ENV WEIXIN_VERSION=1.0.3

WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["hatchery"]
