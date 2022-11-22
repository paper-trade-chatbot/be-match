FROM alpine:latest

RUN apk add --update-cache tzdata
COPY be-match /be-match

ENTRYPOINT ["/be-match"]


