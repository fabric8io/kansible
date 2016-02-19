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
    def tag = 'latest'

    kubernetes.image().withName(imageName).build().fromPath(".")
    kubernetes.image().withName(imageName).tag().inRepository('docker.io/fabric8/'+image).withTag(tag)
    kubernetes.image().withName('docker.io/fabric8/'+image).push().withTag(tag).toRegistry()

  }
}
