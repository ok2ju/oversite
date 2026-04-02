FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod ./
# go.sum may not exist yet; glob pattern copies it only if present
COPY go.su[m] ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o oversite ./cmd/oversite

FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /build/oversite .
COPY --from=builder /build/migrations ./migrations

EXPOSE 8080 8081

ENTRYPOINT ["./oversite"]
