# Stage 1: Build the Go application
FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api .

# Stage 2: Run the application
FROM alpine:3.20

RUN apk --no-cache add ca-certificates
RUN adduser -D -H appuser

WORKDIR /app

COPY --from=build /out/api /app/api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
