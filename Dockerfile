FROM ubuntu:trusty
MAINTAINER GoCD Team <go-cd@googlegroups.com>

RUN apt-get update
RUN apt-get -y upgrade

ADD gogoagent goagent
ENTRYPOINT ["/goagent"]
