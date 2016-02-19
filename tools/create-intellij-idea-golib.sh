#!/bin/bash

# Build up a GOPATH directory for IntelliJ IDEA
# which doesn't support GO15VENDOREXPERIMENT yet


pushd `dirname $0`/.. > /dev/null
base=`pwd`
popd > /dev/null
golib=${base}/golib

rm -rf ${golib}

# Link all from the vendor dirs pulled by glide:
vendor_src=${golib}/vendor/src
mkdir -p ${vendor_src}
for f in ${base}/vendor/*
do
  # echo "Symlinking vendor source dir: ${base}/${f}"
  ln -s "${f}" "${vendor_src}/"
done

# Link self into the golib dir
self_src=${golib}/self/src/github.com/fabric8io
mkdir -p ${self_src}
ln -s "${base}" "${self_src}/kansible";

echo "Use the following dirs exclusively as go-libraries in  IntelliJ IDEA:"
echo "(Preferences -> Languages & Frameworks -> Go -> Go Libraries, Add to Project Libraries, uncheck 'use system defined GOPATH')"
echo
echo "${golib}/vendor"
echo "${golib}/self"
