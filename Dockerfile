FROM debian:stable
MAINTAINER Ronoaldo JLP <ronoaldo@gmail.co>

RUN apt-get update -q && apt-get -y install ca-certificates && apt-get clean
RUN useradd -ms /bin/bash discord
USER discord

ARG GIT_HASH
ENV BOT_VERSION $GIT_HASH
ADD discordbot /usr/bin/discordbot

ENTRYPOINT /usr/bin/discordbot
