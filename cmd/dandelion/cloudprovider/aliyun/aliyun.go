package cloudprovideraliyun

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/tengattack/dandelion/log"
)

const upperhex = "0123456789ABCDEF"

func shouldEscape(c byte) bool {
	// §2.3 Unreserved characters (alphanum)
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}
	switch c {
	case '-', '_', '.', '~': // §2.3 Unreserved characters (mark)
		return false
	}
	return true
}

func escape(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			hexCount++
		}
	}

	if hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case shouldEscape(c):
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

// Base64 base64
func Base64(src []byte) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}

// HmacSha1 hmac sha1
func HmacSha1(secretKey string, str string) []byte {
	h := hmac.New(sha1.New, []byte(secretKey))
	_, _ = h.Write([]byte(str))
	return h.Sum(nil)
}

const (
	endpoint   = "https://ecs.aliyuncs.com/"
	apiVersion = "2014-05-26"
)

type AliyunEcsCommonParams struct {
	AccessKeyID  string
	AccessSecret string
	RegionID     string
}

func sign(common AliyunEcsCommonParams, method, endpoint string, params map[string]string) url.Values {
	params["Version"] = apiVersion
	params["Format"] = "JSON"
	params["AccessKeyId"] = common.AccessKeyID
	params["SignatureNonce"] = uuid.New().String()
	// 2018-01-01T12:00:00Z
	ts := time.Now().UTC().Format(time.RFC3339)
	params["Timestamp"] = ts
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureVersion"] = "1.0"

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	body := make(url.Values, len(keys)+1)

	var canonicalizedQueryString string
	for i, k := range keys {
		if i > 0 {
			canonicalizedQueryString += "&"
		}
		canonicalizedQueryString += escape(k) + "=" + escape(params[k])
		body.Set(k, params[k])
	}

	strToSign :=
		method + "&" +
			escape("/") + "&" +
			escape(canonicalizedQueryString)

	signature := Base64(HmacSha1(common.AccessSecret+"&", strToSign))
	body.Set("Signature", signature)
	return body
}

func request(common AliyunEcsCommonParams, action string, params map[string]string, v interface{}) error {
	reqParams := make(map[string]string, len(params)+2)
	reqParams["Action"] = action
	reqParams["RegionId"] = common.RegionID
	for k, v := range params {
		reqParams[k] = v
	}
	q := sign(common, "GET", endpoint, reqParams)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	log.LogAccess.Debugf("aliyun request: %s", endpoint+"?"+q.Encode())
	req, err := http.NewRequest(http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	log.LogAccess.Debugf("aliyun response: %s", body)
	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

type AliyunEcsDescribeInstance struct {
	InstanceId string `json:"InstanceId"`
	ZoneId     string `json:"ZoneId"`
	RegionId   string `json:"RegionId"`

	Cpu    int `json:"Cpu"`
	Memory int `json:"Memory"`

	InstanceNetworkType string `json:"InstanceNetworkType"`
	VpcAttributes       struct {
		PrivateIpAddress struct {
			IpAddress []string `json:"IpAddress"`
		} `json:"PrivateIpAddress"`
	} `json:"VpcAttributes"`
}

type AliyunEcsDescribeInstancesResp struct {
	RequestID  string `json:"RequestId"`
	PageNumber int    `json:"PageNumber"`
	PageSize   int    `json:"PageSize"`
	NextToken  string `json:"NextToken"`
	TotalCount int64  `json:"TotalCount"`

	Instances struct {
		Instance []AliyunEcsDescribeInstance `json:"Instance"`
	} `json:"Instances"`
}

type AliyunEcsModifyInstanceAttributeResp struct {
	RequestID string `json:"RequestId"`
}

// SetNodeName set aliyun ecs instance name
func SetNodeName(cp map[string]string, ip, nodeName string) error {
	var common AliyunEcsCommonParams
	common.AccessKeyID = cp["access_key_id"]
	common.AccessSecret = cp["access_secret"]
	common.RegionID = cp["region_id"]

	var resp AliyunEcsDescribeInstancesResp
	err := request(common, "DescribeInstances", map[string]string{
		"PrivateIpAddresses": fmt.Sprintf(`["%s"]`, ip),
	}, &resp)
	if err != nil {
		return err
	}
	if len(resp.Instances.Instance) == 1 {
		// matched
		var modifyResp AliyunEcsModifyInstanceAttributeResp
		ins := resp.Instances.Instance[0]
		err := request(common, "ModifyInstanceAttribute", map[string]string{
			"InstanceId":   ins.InstanceId,
			"InstanceName": nodeName,
		}, &modifyResp)
		if err != nil {
			return err
		}
	}
	return nil
}
