#!/usr/bin/groovy
node{
  stage 'canary release'
  git 'https://github.com/fabric8io/kansible.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder').inside {

    retry(3){
      sh 'make bootstrap'
    }

    retry(3){
      sh "cd /go/src/workspace/${env.JOB_NAME} && make build test lint"
    }

    def imageName = 'kansible'
    def tag = 'latest'

    kubernetes.image().withName(imageName).build().fromPath(".")
    kubernetes.image().withName(imageName).tag().inRepository('docker.io/fabric8/'+imageName).force().withTag(tag)
    kubernetes.image().withName('docker.io/fabric8/'+imageName).push().withTag(tag).toRegistry()

  }
}
