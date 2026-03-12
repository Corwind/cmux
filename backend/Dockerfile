# Stage 1: Build
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cmux ./cmd/cmux

# Stage 2: Production
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /app/cmux .
RUN mkdir -p /app/db
EXPOSE 8080
CMD ["./cmux"]
