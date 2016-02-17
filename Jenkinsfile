node{
  stage 'canary release'
  git 'https://github.com/fabric8io/gosupervise.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder').inside {
    sh 'make bootstrap'
    sh "cd /go/src/workspace/${env.JOB_NAME} && make build"

    def imageName = 'fabric8/gosupervise'
    def newVersion = getNewVersion{}
    kubernetes.image().withName(imageName).build().fromPath(".")
    kubernetes.image().withName(imageName).tag().inRepository("${env.FABRIC8_DOCKER_REGISTRY_SERVICE_HOST}:${env.FABRIC8_DOCKER_REGISTRY_SERVICE_PORT}/${env.KUBERNETES_NAMESPACE}/${imageName}").withTag(newVersion)
    kubernetes.image().withName("${env.FABRIC8_DOCKER_REGISTRY_SERVICE_HOST}:${env.FABRIC8_DOCKER_REGISTRY_SERVICE_PORT}/${env.KUBERNETES_NAMESPACE}/${imageName}").push().withTag(newVersion).toRegistry()
  }
}
