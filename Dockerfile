FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -trimpath  -o /out/meshchat-server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /srv/meshchat
COPY --from=builder /out/meshchat-server /usr/local/bin/meshchat-server

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/meshchat-server"]
