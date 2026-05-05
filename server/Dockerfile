FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o sovereignstream-server ./main.go

FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/sovereignstream-server ./sovereignstream-server
EXPOSE 8080 1935 3000
ENTRYPOINT ["/app/sovereignstream-server"]
