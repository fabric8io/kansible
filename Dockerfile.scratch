FROM scratch
# Call 'make build-all' before building it
ADD bin/kansible-docker /kansible
# Variables are interpolated not by Docker but by kansible (they are transmitted literally)
CMD [ "/kansible", "pod", "$KANSIBLE_HOSTS"]
