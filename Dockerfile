FROM golang:1.23

WORKDIR /usr/src/app

COPY go.mod ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o /usr/local/bin/app

CMD ["app"]