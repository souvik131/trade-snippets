FROM golang:1.23-alpine3.21 AS build

RUN apk add build-base
WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=1 go build -o /usr/local/bin/app


FROM alpine:3.17
RUN apk add --no-cache tzdata
ENV TZ=Asia/Kolkata
WORKDIR /
COPY --from=0 /usr/local/bin/app .
COPY --from=build /usr/src/app/.env .env
CMD ["/app"]
