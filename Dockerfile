FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /sysmon-daemon ./cmd/daemon

FROM alpine:3.19
COPY --from=builder /sysmon-daemon /sysmon-daemon
ENTRYPOINT ["/sysmon-daemon"]
