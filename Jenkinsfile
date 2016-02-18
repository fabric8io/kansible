#!/usr/bin/groovy
node{
  stage 'canary release'
  git 'https://github.com/fabric8io/kansible.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder').withPrivileged(true)
      .withHostPathMount('/var/run/docker.sock','/var/run/docker.sock')
      .withEnvVar('DOCKER_CONFIG','/home/jenkins/.docker/')
      .withSecret('jenkins-docker-cfg','/home/jenkins/.docker')
      .withServiceAccount('jenkins').inside {

    sh 'make bootstrap'
    sh "cd /go/src/workspace/${env.JOB_NAME} && make build"

    def imageName = 'kansible'
    def newVersion = getNewVersion{}

    sh "docker build --rm -t ${imageName} ."
    sh "docker tag -f ${imageName} docker.io/fabric8/${imageName}"
    sh "docker push docker.io/fabric8/${imageName}"
  }
}
