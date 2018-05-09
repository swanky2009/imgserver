FROM centos

MAINTAINER swanky <swanky2009@qq.com>

COPY . /usr/imgserver

WORKDIR /usr/imgserver

RUN ["chmod", "+x", "/usr/imgserver/imgserver"]

VOLUME ["/var/upload"]

RUN ["chmod", "+w", "-R", "/var/upload"]

EXPOSE 2300 2301

ENTRYPOINT ["/usr/imgserver/imgserver"]

CMD ["-config=imgserver.cfg"]
