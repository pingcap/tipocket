FROM alpine:3.9

RUN apk update && apk add --no-cache wget bash sed
COPY ./bin/cross-region /bin/cross-region
ENTRYPOINT [ "/bin/cross-region" ]
