package cloudprovider

import (
	"fmt"

	aliyun "github.com/tengattack/dandelion/cmd/dandelion/cloudprovider/aliyun"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
)

// SetNodeName set cloud instance node name
func SetNodeName(ip, nodeName string) error {
	if config.Conf.CloudProvider == nil {
		return nil
	}
	cp := config.Conf.CloudProvider
	provider, ok := cp["provider"]
	if !ok {
		return nil
	}
	switch provider {
	case "aliyun":
		return aliyun.SetNodeName(cp, ip, nodeName)
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}
