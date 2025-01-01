FROM golang:1.23.0 AS builder

WORKDIR /app/

COPY go.* ./

RUN go mod download

COPY cmd/ cmd/
COPY config/ config/
COPY internal/ internal/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o vault-secret-syncer cmd/secret-syncer/main.go

FROM alpine:3.20.2
WORKDIR /app
COPY --from=builder /app/vault-secret-syncer .

EXPOSE 8080
ENTRYPOINT ["/app/vault-secret-syncer"]




