FROM golang:1.24.3 AS builder
ARG VERSION="unknow"

WORKDIR /app/

COPY go.* ./

RUN go mod download

COPY cmd/ cmd/
COPY config/ config/
COPY internal/ internal/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-X main.version=$VERSION -X main.buildTime=$(date +%Y-%m-%d-%H:%M:%S)" -o vault-secret-syncer cmd/secret-syncer/main.go

FROM alpine:3.20.2
WORKDIR /app
COPY --from=builder /app/vault-secret-syncer .

EXPOSE 8080
ENTRYPOINT ["/app/vault-secret-syncer"]




