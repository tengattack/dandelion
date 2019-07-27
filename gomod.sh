#!/bin/sh

gomod() {
  cd /go/src/$1

  GO111MODULE=on GOPROXY=https://goproxy.io go mod init
  echo -e '\nrequire (\n\tgithub.com/confluentinc/confluent-kafka-go v0.11.6\n)\n' >> go.mod
  GO111MODULE=on GOPROXY=https://goproxy.io go mod vendor
  rsync -a ./vendor/ /go/src/
  rm -rf ./vendor go.mod go.sum
}

gomod github.com/tengattack/dandelion
