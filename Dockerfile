FROM centos:7

WORKDIR /playbook

ENV PATH $PATH:/usr/local/gosupervise/

ADD ./bin/gosupervise /usr/local/gosupervise/

CMD gosupervise pod $GOSUPERVISE_HOSTS $GOSUPERVISE_COMMAND
