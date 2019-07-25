# This image expects to find configuration file under /home/postmail/etc/config.yaml
FROM centos:7

RUN groupadd postmail && useradd -g  postmail postmail
USER postmail
WORKDIR /home/postmail

COPY postmail /home/postmail
RUN mkdir /home/postmail/etc
CMD ./postmail --cfg etc/config.yaml

