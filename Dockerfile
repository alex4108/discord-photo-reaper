FROM golang:1.21-alpine AS builder  
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go .
COPY go.* .
RUN go build -o /app/discord-photo-reaper

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/discord-photo-reaper /app/discord-photo-reaper 
RUN chmod +x /app/discord-photo-reaper 
ENTRYPOINT [ "/app/discord-photo-reaper" ]
