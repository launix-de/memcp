# syntax=docker/dockerfile:1

FROM ubuntu:24.04

WORKDIR /memcp
VOLUME /data
EXPOSE 4332
EXPOSE 3307

RUN apt-get update
RUN apt-get install -y software-properties-common
RUN add-apt-repository -y ppa:longsleep/golang-backports
RUN apt-get -y install golang git
RUN git clone https://github.com/launix-de/memcp .
RUN go get
RUN go build
RUN apt-get -y purge golang software-properties-common git
RUN apt-get -y autoremove

ENV PARAMS=
CMD ./memcp -data /data ${PARAMS}
#CMD update cd /app && killall memcp && go build && ./memcp
