FROM centos:7

WORKDIR /playbook

ENV PATH $PATH:/usr/local/kansible/

ADD ./bin/kansible /usr/local/kansible/

CMD kansible pod $KANSIBLE_HOSTS
