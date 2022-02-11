FROM golang:1.17-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY app app

RUN go build -o veryfreshpod

FROM alpine:3.15

WORKDIR /app

COPY --from=builder /build/veryfreshpod .

ENTRYPOINT ["./veryfreshpod"]
