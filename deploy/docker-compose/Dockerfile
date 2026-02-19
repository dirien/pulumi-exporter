FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /pulumi-exporter .

FROM cgr.dev/chainguard/static:latest
COPY --from=builder /pulumi-exporter /usr/bin/pulumi-exporter
ENTRYPOINT ["/usr/bin/pulumi-exporter"]
