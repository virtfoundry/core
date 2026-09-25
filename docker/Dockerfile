FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -H -u 65532 nonroot
WORKDIR /app
COPY --from=builder /server ./server
ENV CONFIG_PATH=/app/config/config.yaml
USER 65532:65532
EXPOSE 8080
CMD ["./server"]
