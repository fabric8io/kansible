#!/bin/bash
rm -rf vendor/src
mkdir vendor/src
FILES=vendor/*
for f in $FILES
do
  if [ "$f" != "vendor/src" ]
  then
    FULL=`pwd`/$f
    echo "Symlinking vendor source dir: $FULL"
    ln -s $FULL vendor/src
  fi
done
