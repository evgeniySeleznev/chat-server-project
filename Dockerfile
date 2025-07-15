FROM golang:1.24.5-alpine3.22 AS builder_chat

COPY . /github.com/evgeniySeleznev/chat-server-project/source
WORKDIR /github.com/evgeniySeleznev/chat-server-project/source

RUN go mod download
RUN go build -o ./bin/chat_server cmd/grpc_server/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder_chat /github.com/evgeniySeleznev/chat-server-project/source/bin/chat_server .

CMD ["./chat_server"]