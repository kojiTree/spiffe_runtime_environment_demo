FROM golang:1.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY demo/server ./demo/server

RUN CGO_ENABLED=0 go build -o /out/demo-server ./demo/server

FROM gcr.io/distroless/static-debian12
WORKDIR /opt/demo
COPY --from=builder /out/demo-server ./demo-server

ENTRYPOINT ["/opt/demo/demo-server"]
