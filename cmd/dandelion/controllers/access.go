package controllers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/log"
)

// TableNameAccessCheck app configs table
func TableNameAccessCheck() string {
	return config.Conf.Database.TablePrefix + "dandelion_accesscheck"
}

type AccessCheckItem struct {
	ID     int64  `db:"id" json:"id"`
	Status int    `db:"status" json:"status"`
	Type   int    `db:"type" json:"type"`
	IPCidr string `db:"ip_cidr" json:"ip_cidr"`

	ipnet *net.IPNet
}

func (item *AccessCheckItem) AllowIP(ip net.IP) bool {
	if item.ipnet == nil {
		return false
	}
	return item.ipnet.Contains(ip)
}

// AccessCheckResp .
type AccessCheckResp struct {
	IP bool `json:"ip"`
}

type AccessChecker struct {
	items   []AccessCheckItem
	expires time.Time
	mu      sync.Mutex
}

var accessChecker = &AccessChecker{}

// isValid returns valid and stale
func (c *AccessChecker) isValid() (valid bool, stale bool) {
	if c.items == nil {
		return false, false
	}
	return true, time.Now().After(c.expires)
}

func (c *AccessChecker) AllowIP(ip net.IP) bool {
	accessChecker.mu.Lock()
	defer accessChecker.mu.Unlock()

	ipAllow := false
	for i := range c.items {
		if c.items[i].AllowIP(ip) {
			ipAllow = true
			break
		}
	}
	return ipAllow
}

func getAccessChecker() (*AccessChecker, error) {
	accessChecker.mu.Lock()
	defer accessChecker.mu.Unlock()

	valid, stale := accessChecker.isValid()
	if valid && !stale {
		return accessChecker, nil
	}

	var items []AccessCheckItem
	err := config.DB.Select(&items, "SELECT * FROM "+TableNameAccessCheck()+" WHERE type = 1 AND status = 1 ORDER BY id ASC")
	if err != nil {
		if valid {
			log.LogError.Error(err)
			// PASS
			return accessChecker, nil
		}
		return nil, err
	}
	validItems := make([]AccessCheckItem, 0, len(items))
	for i := range items {
		_, ipnet, err := net.ParseCIDR(items[i].IPCidr)
		if err != nil {
			log.LogError.WithField("id", items[i].ID).Errorf("invalid ip cidr: %s", items[i].IPCidr)
			// PASS
			continue
		}
		item := items[i]
		item.ipnet = ipnet
		validItems = append(validItems, item)
	}
	accessChecker.items = validItems
	accessChecker.expires = time.Now().Add(5 * time.Minute)
	return accessChecker, nil
}

func accessCheckHandler(c *gin.Context) {
	ipStr, _ := c.GetQuery("ip")
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		abortWithError(c, http.StatusBadRequest, "params error")
		return
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		abortWithError(c, http.StatusBadRequest, "params error")
		return
	}

	checker, err := getAccessChecker()
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	ipAllow := checker.AllowIP(ip)

	succeed(c, gin.H{"ip": ipAllow})
}
