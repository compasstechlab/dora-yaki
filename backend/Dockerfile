# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Production stage
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /server /app/server
ENV TZ=Asia/Tokyo
EXPOSE 7202
CMD ["/app/server"]
