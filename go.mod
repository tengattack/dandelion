module github.com/tengattack/dandelion

go 1.15

require (
	github.com/confluentinc/confluent-kafka-go v1.5.2
	github.com/gin-gonic/gin v1.6.3
	github.com/go-git/go-git v4.7.0+incompatible
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/googleapis/gnostic v0.0.0-00010101000000-000000000000 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tengattack/logrus-agent-hook v0.0.0-20190205052644-35fd90001636 // indirect
	github.com/tengattack/tgo v0.0.0-20190420105756-400cce13925e
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
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
