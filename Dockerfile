# Stage 1: Build
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG BUILD=unknown
RUN CGO_ENABLED=0 go build \
    -ldflags="-X github.com/pltanton/lsbot/internal/mcp.ServerVersion=${VERSION} -X main.Build=${BUILD} -w -s" \
    -o /lsbot .

# Stage 2: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /lsbot /usr/local/bin/lsbot
ENTRYPOINT ["lsbot"]
CMD ["router"]
