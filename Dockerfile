FROM golang:1.15-alpine3.12

ARG version
ARG proxy
ARG goproxy

# Download packages from aliyun mirrors
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --update add --no-cache ca-certificates tzdata

COPY . /go/src/github.com/tengattack/dandelion
RUN cd /go/src/github.com/tengattack/dandelion \
#  && apk --update add --no-cache git openssh-client rsync build-base openssl-dev zlib-dev librdkafka-dev
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

FROM alpine:3.12

# Download packages from aliyun mirrors
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk --update add --no-cache ca-certificates tzdata openssl zlib git openssh-client \
  # using /etc/hosts over DNS
  # https://github.com/golang/go/issues/35305
  && echo "hosts: files dns" > /etc/nsswitch.conf

COPY --from=0 /go/bin/dandelion /go/bin/dandelion-seed /bin/
