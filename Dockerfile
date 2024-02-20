# syntax=docker/dockerfile:1

FROM ubuntu:22.04
WORKDIR /app
COPY . .
RUN apt-get update
RUN apt-get install -y software-properties-common
RUN add-apt-repository -y ppa:longsleep/golang-backports
RUN apt-get -y install golang
RUN go get
RUN go build
#RUN apt-get purge golang
CMD ./memcp
#CMD update cd /app && killall memcp && go build && ./memcp
