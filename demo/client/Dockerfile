FROM golang:1.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY demo/client ./demo/client

RUN CGO_ENABLED=0 go build -o /out/demo-client ./demo/client

FROM gcr.io/distroless/static-debian12
WORKDIR /opt/demo
COPY --from=builder /out/demo-client ./demo-client

ENTRYPOINT ["/opt/demo/demo-client"]
