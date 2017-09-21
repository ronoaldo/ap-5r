FROM debian:stable
MAINTAINER Ronoaldo JLP <ronoaldo@gmail.co>

RUN apt-get update -q && apt-get -y install ca-certificates && apt-get clean
ADD discordbot /usr/bin/discordbot

RUN useradd -ms /bin/bash discord

USER discord
ENTRYPOINT /usr/bin/discordbot
