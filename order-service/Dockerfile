FROM golang:1.23-alpine as build

RUN apk update
RUN apk add git

WORKDIR /app

COPY go.mod /app/
COPY go.sum /app/

RUN go mod download
RUN go mod tidy

COPY . /app/

WORKDIR /app/cmd/webservice

RUN go build -o /app/main

# --------

FROM alpine:3.18

WORKDIR /app

# Web service
EXPOSE 8080

COPY --from=build /app/main /app/main
# COPY --from=build /app/asset /app/asset

CMD ["./main"]