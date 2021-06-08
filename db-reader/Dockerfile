FROM golang:alpine

ENV TZ=Etc/UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /src
RUN adduser -D user && chown -R user /src
COPY ./generator .
COPY ./db-reader .
COPY ./common .

USER user

ENTRYPOINT [ "./run-with-reborn.sh" ]
