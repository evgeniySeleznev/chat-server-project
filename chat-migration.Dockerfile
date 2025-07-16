FROM alpine:3.22.1

RUN apk update && \
    apk upgrade && \
    apk add bash && \
    rm -rf /var/cache/apk/*

ADD https://github.com/pressly/goose/releases/download/v3.14.0/goose_linux_x86_64 /bin/goose
RUN chmod +x /bin/goose

WORKDIR /root

ADD migrations/*.sql migrations/
ADD chat-migration.sh .
ADD .env .

RUN ls -la /root

RUN chmod +x chat-migration.sh

ENTRYPOINT ["bash", "chat-migration.sh"]