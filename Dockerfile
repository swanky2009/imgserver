FROM centos

MAINTAINER swanky <swanky2009@qq.com>

COPY ./app /usr/imgserver

WORKDIR /usr/imgserver

VOLUME ["/var/upload"]

EXPOSE 2300 2301

ENTRYPOINT ["imgserver"]

CMD ["-config","imgserver.cfg"]