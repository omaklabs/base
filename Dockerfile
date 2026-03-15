FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate
RUN go build -o bin/app ./cmd/app

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/app .
EXPOSE 8080
CMD ["./app"]
