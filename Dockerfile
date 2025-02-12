FROM golang:1.23-alpine3.21 AS build

RUN apk add build-base
WORKDIR /usr/src/app

# Pre-copy/cache go.mod for efficient dependency management
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=1 go build -o /usr/local/bin/app


FROM alpine:3.17
RUN apk add --no-cache tzdata
ENV TZ=Asia/Kolkata

WORKDIR /app
COPY --from=build /usr/local/bin/app .
COPY --from=build /usr/src/app/.env .env

# Ensure the binary directory exists and is writable
RUN mkdir -p /app/binary && chmod -R 777 /app/binary

VOLUME ["/app/binary"]

CMD ["./app"]

