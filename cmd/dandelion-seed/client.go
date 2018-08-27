package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/cmd/dandelion-seed/config"
	"github.com/tengattack/dandelion/log"
)

// errors
var (
	ErrFileIsOccupiedByDir = errors.New("config file is occupied by the directory")
)

var (
	metadataRegexp *regexp.Regexp
)

func init() {
	metadataRegexp = regexp.MustCompile(`"?(version|host|instance_id)"?\s*[=:]\s*"?(\S+?)["\s$]`)
}

// ReadMetadataFromFile read metadata to client config from file
func ReadMetadataFromFile(appID, appPath string, metaFiles []string) (*app.ClientConfig, error) {
	hostname, _ := os.Hostname()
	cfg := app.ClientConfig{
		AppID:      appID,
		Host:       hostname,
		InstanceID: hostname,
		Version:    "0",
	}
	for _, metaFile := range metaFiles {
		filePath := path.Join(appPath, metaFile)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		s := metadataRegexp.FindAllStringSubmatch(string(data), -1)
		for _, s0 := range s {
			switch s0[1] {
			case "host":
				cfg.Host = s0[2]
			case "instance_id":
				cfg.InstanceID = s0[2]
			case "version":
				cfg.Version = s0[2]
			}
		}
	}
	log.LogAccess.Debugf("[%s] client config: %v", appID, cfg)
	return &cfg, nil
}

// ResyncConfigFiles sync config files
func ResyncConfigFiles(appConfig *config.SectionConfig, c *app.AppConfig, files []string) error {
	log.LogAccess.Infof("[%s] resyncing config files", c.AppID)
	var uid, gid int
	var mode os.FileMode
	if appConfig.Chown != "" {
		parts := strings.Split(appConfig.Chown, ":")
		u, err := user.Lookup(parts[0])
		if err != nil {
			log.LogError.Errorf("[%s] failed to lookup user '%s': %v", c.AppID, parts[0], err)
			return err
		}
		uid, _ = strconv.Atoi(u.Uid)
		if len(parts) > 1 {
			g, err := user.LookupGroup(parts[1])
			if err != nil {
				log.LogError.Errorf("[%s] failed to lookup group '%s': %v", c.AppID, parts[1], err)
				return err
			}
			gid, _ = strconv.Atoi(g.Gid)
		} else {

			gid, _ = strconv.Atoi(u.Gid)
		}
	}
	if appConfig.Chmod != "" {
		modeVal, _ := strconv.ParseInt(appConfig.Chmod, 8, 32)
		mode = os.FileMode(modeVal)
	}
	for _, file := range files {
		filePath := path.Join(appConfig.Path, file)
		err := Client.Download(c.AppID, c.CommitID, file, filePath)
		if err != nil {
			return err
		}
		if uid != 0 {
			err := os.Chown(filePath, uid, gid)
			if err != nil {
				log.LogError.Errorf("[%s] failed to change ownership for file '%s': %v", c.AppID, filePath, err)
				return err
			}
		}
		if mode != 0 {
			err := os.Chmod(filePath, mode)
			if err != nil {
				log.LogError.Errorf("[%s] failed to change permission for file '%s': %v", c.AppID, filePath, err)
				return err
			}
		}
	}
	if appConfig.ExecReload != "" {
		parts := strings.Fields(appConfig.ExecReload)
		out, err := exec.Command(parts[0], parts[1:]...).Output()
		if len(out) > 0 {
			log.LogAccess.Infof("[%s] exec reload:\n%s", c.AppID, string(out))
		} else {
			log.LogAccess.Infof("[%s] exec reload", c.AppID)
		}
		if err != nil {
			log.LogError.Errorf("[%s] exec reload error: %v", c.AppID, err)
			return err
		}
	}
	return nil
}

// CheckAppConfig check single app's config
func CheckAppConfig(appConfig *config.SectionConfig) error {
	log.LogAccess.Debugf("[%s] checking", appConfig.AppID)
	clientConfig, err := ReadMetadataFromFile(appConfig.AppID, appConfig.Path, appConfig.MetaFiles)
	if err != nil {
		log.LogError.Errorf("[%s] read metadata error: %v", appConfig.AppID, err)
		return err
	}
	c, err := Client.Match(clientConfig)
	if err != nil {
		log.LogError.Errorf("[%s] match error: %v", appConfig.AppID, err)
		return err
	}
	files, err := Client.ListFiles(c.AppID, c.CommitID)
	if err != nil {
		log.LogError.Errorf("[%s] list files error: %v", c.AppID, err)
		return err
	}
	dirty := false
	h := md5.New()
	for _, file := range files {
		filePath := path.Join(appConfig.Path, file)
		s, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			dirty = true
			log.LogAccess.Infof("[%s] config file lost", c.AppID)
			break
		}
		if err != nil {
			return err
		}
		if s.IsDir() {
			return ErrFileIsOccupiedByDir
		}
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		io.Copy(h, f)
	}
	if !dirty {
		md5sum := hex.EncodeToString(h.Sum(nil))
		if md5sum != c.MD5Sum {
			dirty = true
			log.LogAccess.Infof("[%s] config file md5sum mismatch: \"%s\" != \"%s\"", c.AppID, md5sum, c.MD5Sum)
		}
	}
	if dirty {
		// Sync config
		err = ResyncConfigFiles(appConfig, c, files)
		if err != nil {
			log.LogError.Errorf("[%s] resync config files error: %v", c.AppID, err)
			return err
		}
	}
	return nil
}

// CheckCurrentConfigs check configs
func CheckCurrentConfigs() error {
	for _, config := range Conf.Configs {
		err := CheckAppConfig(&config)
		if err != nil {
			return err
		}
	}
	return nil
}
