FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY app ./app

RUN go build -o veryfreshpod

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /build/veryfreshpod .

ENTRYPOINT ["/app/veryfreshpod"]
