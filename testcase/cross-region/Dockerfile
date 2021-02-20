FROM alpine:3.9

RUN apk update && apk add --no-cache wget bash sed
COPY ./bin/cross-region-linux /bin/cross-region-linux
ENTRYPOINT [ "/bin/cross-region-linux" ]
