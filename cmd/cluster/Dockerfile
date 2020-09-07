FROM alpine:3.12.0

RUN apk update && apk upgrade && apk add --no-cache bash curl wget

WORKDIR /src
COPY . .

CMD [ "bash", "./mock_workload.sh" ]
