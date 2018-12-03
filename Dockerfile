FROM debian:stable
MAINTAINER Ronoaldo JLP <ronoaldo@gmail.com>

RUN apt-get update -q && apt-get -y install ca-certificates && apt-get clean
RUN useradd -ms /bin/bash discord
USER discord

ARG GIT_HASH
ENV BOT_VERSION $GIT_HASH
ENV BOT_ASSET_DIR "/var/cache/ap-5r/assets"
ENV BOT_CACHE_DIR "/var/cache/ap-5r"

ADD images/characters/* /var/cache/ap-5r/assets/images/characters/
ADD images/ui/* /var/cache/ap-5r/assets/images/ui/
ADD ap-5r /usr/bin/ap-5r

CMD ["/usr/bin/ap-5r"]
