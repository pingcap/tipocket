FROM alpine:3.9

RUN apk update && apk add --no-cache wget bash sed
COPY ./bin/stale-read /bin/stale-read
ENTRYPOINT [ "/bin/stale-read" ]
