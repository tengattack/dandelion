package main

import (
	"archive/zip"
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

	shellwords "github.com/mattn/go-shellwords"
	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/cmd/dandelion-seed/config"
	"github.com/tengattack/dandelion/log"
)

// errors
var (
	ErrFileIsOccupiedByDir   = errors.New("config file is occupied by the directory")
	ErrFileNotFoundInArchive = errors.New("file not found in archive")
)

var (
	metadataRegexp *regexp.Regexp
)

func init() {
	metadataRegexp = regexp.MustCompile(`"?(version|host|instance_id)"?\s*[=:]\s*"?(\S+?)["\s$]`)
}

// ReadMetadataFromFile read metadata to client config from file
func ReadMetadataFromFile(appConfig *config.SectionConfig) (*app.ClientConfig, error) {
	hostname, _ := os.Hostname()
	host := os.Getenv("NODE_NAME")
	if host == "" {
		host = os.Getenv("HOST")
		if host == "" {
			host = hostname
		}
	}
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = hostname
	}
	cfg := app.ClientConfig{
		ID:         appConfig.ID,
		AppID:      appConfig.AppID,
		Host:       host,
		InstanceID: instanceID,
		Version:    "0",
	}
	for _, metaFile := range appConfig.MetaFiles {
		filePath := path.Join(appConfig.Path, metaFile)
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
	log.LogAccess.Debugf("[%s] client config: %v", appConfig.AppID, cfg)
	return &cfg, nil
}

func syncExpectedFile(appID string, except io.Reader, fi os.FileInfo, actualFileName, actualFile string) error {
	f, err := os.Open(actualFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// write new directly
		f, err := os.Create(actualFile)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, except)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
		err = os.Chtimes(actualFile, fi.ModTime(), fi.ModTime())
		if err != nil {
			log.LogAccess.Warnf("[%s] chtimes %s error: %v", appID, actualFileName, err)
		}
		return nil
	}

	h := md5.New()
	_, err = io.Copy(h, f)
	f.Close()
	if err != nil {
		return err
	}
	actualMD5 := hex.EncodeToString(h.Sum(nil))

	// compare md5
	e, err := ioutil.ReadAll(except)
	if err != nil {
		return err
	}
	h = md5.New()
	_, err = h.Write(e)
	if err != nil {
		return err
	}
	exceptMD5 := hex.EncodeToString(h.Sum(nil))

	if actualMD5 != exceptMD5 {
		log.LogAccess.Debugf("[%s] file %s md5 mismatch: \"%s\" != \"%s\"", appID, actualFileName, actualMD5, exceptMD5)

		f, err := os.Create(actualFile)
		if err != nil {
			return err
		}
		_, err = f.Write(e)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
		err = os.Chtimes(actualFile, fi.ModTime(), fi.ModTime())
		if err != nil {
			log.LogAccess.Warnf("[%s] chtimes %s error: %v", appID, actualFileName, err)
		}
		return nil
	}

	return nil
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
	z, err := Client.GetZipArchive(c.AppID, c.CommitID)
	if err != nil {
		return err
	}
	for _, fileName := range files {
		var zf *zip.File
		for _, f := range z.File {
			if f.Name == fileName {
				zf = f
				break
			}
		}
		if zf == nil {
			return ErrFileNotFoundInArchive
		}
		filePath := path.Join(appConfig.Path, fileName)
		err = os.MkdirAll(path.Dir(filePath), os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return err
		}
		fr, err := zf.Open()
		if err != nil {
			return err
		}
		err = syncExpectedFile(c.AppID, fr, zf.FileInfo(), fileName, filePath)
		fr.Close()
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
		parts, err := shellwords.Parse(appConfig.ExecReload)
		if err != nil {
			log.LogError.Errorf("[%s] parse reload command error: %v", c.AppID, err)
			return err
		}
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

func checkConfig(appConfig *config.SectionConfig, clientConfig *app.ClientConfig) (*app.AppConfig, error) {
	Client.SetStatus(clientConfig, client.StatusChecking)
	c, err := Client.Match(clientConfig)
	if err != nil {
		log.LogError.Errorf("[%s] match error: %v", appConfig.AppID, err)
		return nil, err
	}
	files, err := Client.ListFiles(c.AppID, c.CommitID)
	if err != nil {
		log.LogError.Errorf("[%s] list files error: %v", c.AppID, err)
		return c, err
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
			return c, err
		}
		if s.IsDir() {
			return c, ErrFileIsOccupiedByDir
		}
		f, err := os.Open(filePath)
		if err != nil {
			return c, err
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
		Client.SetStatus(clientConfig, client.StatusSyncing, map[string]interface{}{
			"config_id": c.ID,
			"commit_id": c.CommitID,
		})
		// Sync config
		err = ResyncConfigFiles(appConfig, c, files)
		if err != nil {
			log.LogError.Errorf("[%s] resync config files error: %v", c.AppID, err)
			return c, err
		}
	}
	return c, nil
}

// CheckAppConfig check single app's config
func CheckAppConfig(appConfig *config.SectionConfig) error {
	log.LogAccess.Debugf("[%s] checking", appConfig.AppID)
	clientConfig, err := ReadMetadataFromFile(appConfig)
	if err != nil {
		log.LogError.Errorf("[%s] read metadata error: %v", appConfig.AppID, err)
		return err
	}

	var v map[string]interface{}
	c, err := checkConfig(appConfig, clientConfig)
	if c != nil {
		v = map[string]interface{}{
			"config_id": c.ID,
			"commit_id": c.CommitID,
		}
	}
	if err != nil {
		Client.SetStatus(clientConfig, client.StatusError, v)
	} else {
		Client.SetStatus(clientConfig, client.StatusSuccess, v)
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
