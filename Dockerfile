FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/anarchy-api ./cmd/anarchy-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates kubectl
COPY --from=builder /out/anarchy-api /usr/local/bin/anarchy-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/anarchy-api"]
