FROM golang:1.15-alpine3.12

ARG version
ARG proxy
ARG goproxy

# Download packages from aliyun mirrors
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --update add --no-cache ca-certificates tzdata git openssh-client rsync \
    build-base openssl-dev zlib-dev librdkafka-dev

COPY . /go/src/github.com/tengattack/dandelion
RUN cd /go/src/github.com/tengattack/dandelion \
#  && chmod +x gomod.sh && ./gomod.sh \
#  && cd /go/src/github.com/confluentinc && rm -rf confluent-kafka-go \
#  && git clone --branch v0.11.6 --single-branch https://github.com/confluentinc/confluent-kafka-go \
#  && cd confluent-kafka-go/kafka/go_rdkafka_generr \
#  && go build && ./go_rdkafka_generr ../generated_errors.go \
#  && cd /go/src/github.com/tengattack/dandelion \
  && cd cmd/dandelion && GOPROXY=$goproxy go install -ldflags "-X main.Version=$version" && cd ../.. \
  && cd cmd/dandelion-seed && GOPROXY=$goproxy go install -ldflags "-X main.Version=$version" && cd ../..

FROM alpine:3.12

# Download packages from aliyun mirrors
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --update add --no-cache ca-certificates tzdata openssl zlib librdkafka \
  # using /etc/hosts over DNS
  # https://github.com/golang/go/issues/35305
  && echo "hosts: files dns" > /etc/nsswitch.conf

COPY --from=0 /go/bin/dandelion /go/bin/dandelion-seed /bin/
