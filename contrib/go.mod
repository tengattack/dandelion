module github.com/tengattack/dandelion

go 1.15

require (
	github.com/Shopify/sarama v1.29.1
	github.com/bsm/sarama-cluster v2.1.15+incompatible
	github.com/confluentinc/confluent-kafka-go v1.5.2
	github.com/gin-gonic/gin v1.7.3
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mattn/go-shellwords v1.0.12
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/onsi/gomega v1.14.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tengattack/logrus-agent-hook v0.0.0-20230522082256-ff02c4ad429e // indirect
	github.com/tengattack/tgo v0.0.0-20230820131731-a6def8bf0817
	go.uber.org/automaxprocs v1.5.1
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/confluentinc/confluent-kafka-go => github.com/confluentinc/confluent-kafka-go v0.11.6
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190221221350-bfb440be4b87
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
)
