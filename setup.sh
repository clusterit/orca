#!/bin/sh

mkdir src pkg bin
export GOPATH=`pwd`
export PATH=$GOPATH/bin:$PATH
mkdir -p src/github.com/clusterit
cd src/github.com/clusterit
git clone git@github.com:clusterit/orca.git
cd orca
make depends
make all

