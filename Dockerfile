FROM golang:1.19.4-buster

ARG version
ARG proxy
ARG goproxy

# Download packages from aliyun mirrors
RUN set -x \
  && sed -i -e 's#http://deb.debian.org#http://mirrors.aliyun.com#g' \
    -e 's#http://security.debian.org#http://mirrors.aliyun.com#g' \
    /etc/apt/sources.list \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata \
#  && apt-get install -y --no-install-recommends git openssh-client rsync build-base openssl-dev zlib-dev librdkafka-dev \
  && rm -rf /var/lib/apt/lists/*

COPY . /go/src/github.com/tengattack/dandelion
RUN cd /go/src/github.com/tengattack/dandelion \
#  && chmod +x gomod.sh && ./gomod.sh \
#  && cd /go/src/github.com/confluentinc && rm -rf confluent-kafka-go \
#  && git clone --branch v0.11.6 --single-branch https://github.com/confluentinc/confluent-kafka-go \
#  && cd confluent-kafka-go/kafka/go_rdkafka_generr \
#  && go build && ./go_rdkafka_generr ../generated_errors.go \
#  && cd /go/src/github.com/tengattack/dandelion \
  && mv contrib/* . \
  && GOPROXY=$goproxy go mod tidy \
  && cd cmd/dandelion && CGO_ENABLED=0 GOPROXY=$goproxy go install -v -ldflags "-X main.Version=$version" && cd ../.. \
  && cd cmd/dandelion-seed && CGO_ENABLED=0 GOPROXY=$goproxy go install -v -ldflags "-X main.Version=$version" && cd ../..

FROM debian:buster-slim

# Download packages from aliyun mirrors
RUN set -x \
  && sed -i -e 's#http://deb.debian.org#http://mirrors.aliyun.com#g' \
    -e 's#http://security.debian.org#http://mirrors.aliyun.com#g' \
    /etc/apt/sources.list \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata git openssh-client \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p /var/log/dandelion-seed \
  && chmod a+rw /var/log/dandelion-seed \
  # using /etc/hosts over DNS
  # https://github.com/golang/go/issues/35305
  && if [ ! -f /etc/nsswitch.conf ]; then echo "hosts: files dns" > /etc/nsswitch.conf; fi

COPY --from=0 /go/bin/dandelion /go/bin/dandelion-seed /bin/
