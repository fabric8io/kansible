#!/bin/bash

function join { local IFS="$1"; shift; echo "$*"; }

copyright-header \
  --copyright-software Kansible \
  --copyright-holder 'Red Hat' \
  --copyright-year 2016 \
  --copyright-software-description 'Directly orchestrate operating system processes via Kubernetes' \
  -o ./ \
  --license-file header.txt \
  -c headers.yml \
  --add-path $(join : `find . -path ./vendor -prune -o -name '*.go' -print`)
