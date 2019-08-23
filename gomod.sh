#!/bin/sh
set -e

hp() {
  http_proxy=$proxy https_proxy=$proxy no_proxy=localhost,127.0.0.0/8,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,100.64.0.0/10,224.0.0.0/4,240.0.0.0/4,docker00 $@
}

gomod() {
  cd /go/src/$1

  GO111MODULE=on GOPROXY=https://goproxy.io go mod init
  echo -e '\nreplace (\n\tgithub.com/confluentinc/confluent-kafka-go => github.com/confluentinc/confluent-kafka-go v0.11.6\n\tk8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471\n\tk8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190221221350-bfb440be4b87\n\tk8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628\n\tk8s.io/client-go => k8s.io/client-go v10.0.0+incompatible\n)\n' >> go.mod
  GO111MODULE=on GOPROXY=https://goproxy.io go mod vendor
  rsync -a ./vendor/ /go/src/
  rm -rf ./vendor go.mod go.sum
}

gomod github.com/tengattack/dandelion

rm -rf /go/src/golang.org/x/net
hp go get -d -v golang.org/x/net/proxy
