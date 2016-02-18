#!/bin/bash

# Build up a GOPATH directory for IntelliJ IDEA
# which doesn't support GO15VENDOREXPERIMENT yet

rm -rf golib
mkdir -p golib/src
base=$(dirname $(realpath $0))
pkg="github.com/fabric8io/kansible"

# Link all from the vendor dirs pulled by glide:
for f in ${base}/vendor/*
do
  # echo "Symlinking vendor source dir: ${base}/${f}"
  ln -s "${f}" "${base}/golib/src/"
done

# Link project dir into the build up gopath:
target="${base}/golib/src/${pkg}"
# echo "Linking project dir into to ${target}"
rm -rf "${target}"
mkdir -p "${target}"
ln -s "${base}" "${target}"

echo "Use the following dir exclusively as golibrary in  IntelliJ IDEA:"
echo "(Preferences -> Languages & Frameworks -> Go -> Go Libraries, Add to Project Libraries, uncheck 'use system defined GOPATH')"
echo "${base}/golib"