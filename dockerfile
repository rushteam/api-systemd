FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o api-systemd



FROM ubuntu

RUN apt-get update && \
    apt-get install -y systemd systemctl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
# COPY ./api-systemd /app/api-systemd
COPY --from=builder /app/api-systemd /app/api-systemd
RUN chmod +x /app/api-systemd
CMD ["/app/api-systemd"]