FROM golang:1.23-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D appuser
WORKDIR /app
COPY --from=build /server /app/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server"]