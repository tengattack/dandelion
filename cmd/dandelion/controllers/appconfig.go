package controllers

import (
	"archive/zip"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/glob"
	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/log"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CommitAuthor is app config commit author structure
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	When  time.Time `json:"when"`
}

// Commit is app config commit structure
type Commit struct {
	Branch   string       `json:"branch"`
	CommitID string       `json:"commit_id"`
	Message  string       `json:"message"`
	Author   CommitAuthor `json:"author"`
}

const (
	// TableNameConfigs the app configs table
	TableNameConfigs = "dandelion_app_configs"
	// TableNameInstances the app instances table
	TableNameInstances = "dandelion_app_instances"
	// ParamsError the http bad request for error params
	ParamsError = "Params error"
)

var (
	l              *sync.Mutex
	lArchive       *sync.RWMutex
	cachedBranches []string
)

func getBranches(force bool) ([]string, error) {
	if force || cachedBranches == nil {
		rbs, err := config.Repo.Branches()
		if err != nil {
			log.LogError.Errorf("get refs error: %v", err)
			return nil, err
		}
		defer rbs.Close()

		var bs []string
		var b *plumbing.Reference
		for b, err = rbs.Next(); err == nil && b != nil; b, err = rbs.Next() {
			if b.Name().IsBranch() {
				bs = append(bs, b.Name().Short())
			}
		}

		// cached
		cachedBranches = bs
	}
	return cachedBranches, nil
}

func getAppID(branch string) string {
	parts := strings.SplitN(branch, "/", 2)
	return parts[0]
}

func getAppIDs() ([]string, error) {
	// list entries
	branches, err := getBranches(false)
	if err != nil {
		log.LogError.Errorf("get refs error: %v", err)
		return nil, err
	}

	var appIDs []string
	for _, branch := range branches {
		appID := getAppID(branch)
		found := false
		for _, a := range appIDs {
			if a == appID {
				found = true
				break
			}
		}
		if !found {
			appIDs = append(appIDs, appID)
		}
	}
	return appIDs, nil
}

func getAppCommit(branch string, commit *object.Commit) Commit {
	return Commit{
		Branch:   branch,
		CommitID: commit.ID().String(),
		Message:  commit.Message,
		Author: CommitAuthor{
			Name:  commit.Author.Name,
			Email: commit.Author.Email,
			When:  commit.Author.When,
		},
	}
}

func appHealthHandler(c *gin.Context) {
	// check whether lock timeout
	l.Lock()
	defer l.Unlock()
	succeed(c, "success")
}

func appSyncHandler(c *gin.Context) {
	appID := c.Param("app_id")

	l.Lock()
	defer l.Unlock()

	var err error
	if appID != "" {
		branches, err := getBranches(false)
		if err != nil {
			log.LogError.Errorf("get branches error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		for _, branch := range branches {
			if appID == getAppID(branch) {
				err = config.Repo.Pull(branch)
				if err != nil && err != git.NoErrAlreadyUpToDate {
					log.LogError.Errorf("pull error: %v", err)
					abortWithError(c, http.StatusInternalServerError, err.Error())
					return
				}
			}
		}
	} else {
		err = config.Repo.SyncBranches()
		if err != nil {
			log.LogError.Errorf("sync branches error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		_, err = getBranches(true)
		if err != nil {
			log.LogError.Errorf("get branches error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h, err := config.Repo.Head()
	if err != nil {
		log.LogError.Errorf("head error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	appIDs, err := getAppIDs()
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	succeed(c, gin.H{
		"app_ids": appIDs,
		"head": gin.H{
			"name":      h.Name(),
			"app_id":    getAppID(h.Name().Short()),
			"commit_id": h.Hash().String(),
		},
	})
}

func appListHandler(c *gin.Context) {
	l.Lock()
	defer l.Unlock()

	appIDs, err := getAppIDs()
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	succeed(c, gin.H{
		"app_ids": appIDs,
	})
}

func appPublishConfigHandler(c *gin.Context) {
	appID := c.Param("app_id")

	version := c.PostForm("version")
	host := c.PostForm("host")
	instanceID := c.PostForm("instance_id")
	commitID := c.PostForm("commit_id")

	if version == "" || host == "" || instanceID == "" || commitID == "" {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}
	_, err := glob.Compile(host)
	_, err2 := glob.Compile(instanceID)
	if err != nil || err2 != nil {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}

	l.Lock()
	// TODO: unlock when it done
	defer l.Unlock()

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := config.Repo.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		log.LogError.Errorf("get commit error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		log.LogError.Errorf("ls-tree error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	h := md5.New()
	// ... get the files iterator and md5 sum the file
	err = tree.Files().ForEach(func(f *object.File) error {
		if strings.HasPrefix(path.Base(f.Name), ".") {
			// ignore dot files
			return nil
		}
		fr, err := f.Reader()
		if err != nil {
			return err
		}
		defer fr.Close()
		_, err = io.Copy(h, fr)
		return err
	})
	if err != nil {
		log.LogError.Errorf("md5 sum error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	t := time.Now().Unix()
	appConfig := app.AppConfig{
		AppID:       appID,
		Status:      1,
		Version:     version,
		Host:        host,
		InstanceID:  instanceID,
		CommitID:    commit.ID().String(),
		MD5Sum:      hex.EncodeToString(h.Sum(nil)),
		Author:      commit.Author.Name,
		CreatedTime: t,
		UpdatedTime: t,
	}

	r, err := config.DB.NamedExec("INSERT INTO "+TableNameConfigs+
		" (app_id, status, version, host, instance_id, commit_id, md5sum, author, created_time, updated_time)"+
		" VALUES (:app_id, :status, :version, :host, :instance_id, :commit_id, :md5sum, :author, :created_time, :updated_time)", &appConfig)
	if err != nil {
		log.LogError.Errorf("db insert error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	appConfig.ID, err = r.LastInsertId()
	if err != nil {
		log.LogError.Errorf("get last insert id error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if config.MQ != nil {
		m := app.NotifyMessage{
			AppID:  appID,
			Event:  "publish",
			Config: appConfig,
		}
		message, err := json.Marshal(m)
		if err != nil {
			log.LogError.Errorf("encode message error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		err = config.MQ.Publish(string(message))
		if err != nil {
			log.LogError.Errorf("publish message error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	succeed(c, gin.H{
		"app_id": appID,
		// TODO: use the correct branch name instead of appID
		"commit": getAppCommit(appID, commit),
		"config": appConfig,
	})
}

func appRollbackConfigHandler(c *gin.Context) {
	appID := c.Param("app_id")

	id, _ := strconv.ParseInt(c.PostForm("id"), 10, 64)

	if id <= 0 {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}

	var appConfig app.AppConfig
	err := config.DB.Get(&appConfig, "SELECT * FROM "+TableNameConfigs+" WHERE id = ? AND status = 1", id)
	if err == sql.ErrNoRows {
		abortWithError(c, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if appConfig.AppID != appID {
		abortWithError(c, http.StatusForbidden, "config id does not belong to specified app id")
		return
	}

	t := time.Now().Unix()
	_, err = config.DB.Exec("UPDATE "+TableNameConfigs+" SET status = 0, updated_time = ? WHERE id = ?", t, id)
	if err != nil {
		log.LogError.Errorf("db update error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if config.MQ != nil {
		// rollback, notify all nodes
		m := app.NotifyMessage{
			AppID:  appID,
			Event:  "rollback",
			Config: appConfig,
		}
		message, err := json.Marshal(m)
		if err != nil {
			log.LogError.Errorf("encode message error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		err = config.MQ.Publish(string(message))
		if err != nil {
			log.LogError.Errorf("publish message error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	succeed(c, gin.H{
		"app_id": appID,
		"config": appConfig,
	})
}

func appMatchConfigHandler(c *gin.Context) {
	appID := c.Param("app_id")

	version := c.Query("version")
	host := c.Query("host")
	instanceID := c.Query("instance_id")

	var configs []app.AppConfig
	// TODO: apply limit & offset
	err := config.DB.Select(&configs, "SELECT * FROM "+TableNameConfigs+" WHERE app_id = ? AND status = 1 AND version <= ? ORDER BY created_time DESC",
		appID, version)
	if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for _, appConfig := range configs {
		glob1, err1 := glob.Compile(appConfig.Host)
		glob2, err2 := glob.Compile(appConfig.InstanceID)
		if err1 != nil || err2 != nil {
			log.LogAccess.Warnf("config %d host or instance_id glob complie failed", appConfig.ID)
			continue
		}
		if glob1.Match(host) && glob2.Match(instanceID) {
			succeed(c, gin.H{
				"app_id": appID,
				"config": appConfig,
			})
			return
		}
	}

	abortWithError(c, http.StatusNotFound, "not found matched config")
}

func appCheckHandler(c *gin.Context) {
	// check, notify all nodes
	appID := c.Param("app_id")

	if appID == "" {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}

	if config.MQ != nil {
		// TODO: using json.Marshal
		message := fmt.Sprintf(`{"app_id":"%s","event":"%s"}`, appID, "check")
		err := config.MQ.Publish(message)
		if err != nil {
			log.LogError.Errorf("publish message error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	succeed(c, gin.H{
		"app_id": appID,
	})
}

func appListConfigsHandler(c *gin.Context) {
	appID := c.Param("app_id")

	var configs []app.AppConfig
	err := config.DB.Select(&configs, "SELECT * FROM "+TableNameConfigs+" WHERE app_id = ? AND status = 1 ORDER BY created_time DESC",
		appID)
	if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if configs == nil {
		// empty array
		configs = []app.AppConfig{}
	}

	succeed(c, gin.H{
		"app_id":  appID,
		"configs": configs,
	})
}

func appListCommitsHandler(c *gin.Context) {
	appID := c.Param("app_id")

	l.Lock()
	defer l.Unlock()

	wt, err := config.Repo.Worktree()
	if err != nil {
		log.LogError.Errorf("worktree error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	var r []Commit

	branchCount := 0
	branches, err := getBranches(false)
	for _, branch := range branches {
		if appID == getAppID(branch) {
			branchCount++
			err = wt.Checkout(&git.CheckoutOptions{
				Branch: plumbing.ReferenceName("refs/heads/" + branch),
				Force:  true,
			})
			if err != nil {
				log.LogError.Errorf("checkout error: %v", err)
				abortWithError(c, http.StatusInternalServerError, err.Error())
				return
			}

			commits, err := config.Repo.Log(&git.LogOptions{})
			if err != nil {
				log.LogError.Errorf("log error: %v", err)
				abortWithError(c, http.StatusInternalServerError, err.Error())
				return
			}
			for commit, err := commits.Next(); err == nil && commit != nil; commit, err = commits.Next() {
				r = append(r, getAppCommit(branch, commit))
			}
			commits.Close()
		}
	}

	if r != nil && branchCount > 1 {
		sort.Slice(r, func(i, j int) bool {
			return r[i].Author.When.After(r[j].Author.When)
		})
	}

	succeed(c, gin.H{
		"app_id":  appID,
		"commits": r,
	})
}

func appListFilesHandler(c *gin.Context) {
	appID := c.Param("app_id")
	commitID := c.Param("commit_id")

	l.Lock()
	defer l.Unlock()

	/*
		// No need to checkout here
		wt, err := Repo.Worktree()
		if err != nil {
			log.LogError.Errorf("worktree error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		err = wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/" + appID),
			Force:  true,
		})
		if err != nil {
			log.LogError.Errorf("checkout error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	*/

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := config.Repo.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		log.LogError.Errorf("get commit error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	var files []string

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		log.LogError.Errorf("ls-tree error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// ... get the files iterator and print the file
	tree.Files().ForEach(func(f *object.File) error {
		if strings.HasPrefix(path.Base(f.Name), ".") {
			// ignore dot files
			return nil
		}
		files = append(files, f.Name)
		return nil
	})

	succeed(c, gin.H{
		"app_id":    appID,
		"commit_id": commitID,
		"files":     files,
	})
}

func appGetFileHandler(c *gin.Context) {
	// appID := c.Param("app_id")
	commitID := c.Param("commit_id")
	path := c.Param("path")

	// remove the beginning slash
	path = path[1:]

	l.Lock()
	defer l.Unlock()

	/*
		// No need to checkout here
		wt, err := Repo.Worktree()
		if err != nil {
			log.LogError.Errorf("worktree error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		err = wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/" + appID),
			Force:  true,
		})
		if err != nil {
			log.LogError.Errorf("checkout error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	*/

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := config.Repo.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		log.LogError.Errorf("get commit error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		log.LogError.Errorf("ls-tree error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	f, err := tree.File(path)
	if err == object.ErrFileNotFound {
		abortWithError(c, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		log.LogError.Errorf("get file error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	fr, err := f.Reader()
	if err != nil {
		log.LogError.Errorf("get file error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer fr.Close()

	d, err := ioutil.ReadAll(fr)
	if err != nil {
		log.LogError.Errorf("read file error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Data(http.StatusOK, "text/plain", d)
}

func buildArchive(appID, commitID, archiveFilePath string) error {
	lArchive.Lock()
	defer lArchive.Unlock()

	log.LogAccess.Infof("building archive for %s/%s", appID, commitID)

	l.Lock()
	defer l.Unlock()

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := config.Repo.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		log.LogError.Errorf("get commit error: %v", err)
		return err
	}

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		log.LogError.Errorf("ls-tree error: %v", err)
		return err
	}

	z, err := os.Create(archiveFilePath)
	if err != nil {
		return err
	}
	defer z.Close()

	zw := zip.NewWriter(z)
	defer zw.Close()

	// ... get the files iterator and print the file
	err = tree.Files().ForEach(func(f *object.File) error {
		if strings.HasPrefix(path.Base(f.Name), ".") {
			// ignore dot files
			return nil
		}
		fh := &zip.FileHeader{
			Name:               f.Name,
			Method:             zip.Deflate,
			UncompressedSize64: uint64(f.Size),
		}
		fh.SetModTime(commit.Author.When)
		fw, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		fr, err := f.Reader()
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, fr)
		return err
	})

	return err
}

func appGetArchiveHandler(c *gin.Context) {
	appID := c.Param("app_id")
	commitID := c.Param("commit_id")

	if !strings.HasSuffix(commitID, ".zip") {
		abortWithError(c, http.StatusBadRequest, "unsupported archive type")
		return
	}
	commitID = commitID[0 : len(commitID)-4]

	archivePath := path.Join(config.Conf.Core.ArchivePath, appID)
	err := os.MkdirAll(archivePath, os.ModePerm)
	if err != nil {
		log.LogError.Errorf("mkdirp error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	archiveFilePath := path.Join(archivePath, commitID+".zip")
	// get archive
	lArchive.RLock()
	f, err := os.Open(archiveFilePath)
	lArchive.RUnlock()

	if err != nil && os.IsNotExist(err) {
		// build archive
		err = buildArchive(appID, commitID, archiveFilePath)
		if err != nil {
			log.LogError.Errorf("build archive error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		lArchive.RLock()
		f, err = os.Open(archiveFilePath)
		lArchive.RUnlock()
	}

	if err != nil {
		log.LogError.Errorf("get archive error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		log.LogError.Errorf("read archive file error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", d)
}

func appListInstancesHandler(c *gin.Context) {
	appID := c.Param("app_id")

	var statuses []app.Status
	// show active instances from last day
	t := time.Now().AddDate(0, 0, -1).Unix()
	err := config.DB.Select(&statuses, "SELECT * FROM "+TableNameInstances+" WHERE app_id = ? AND updated_time >= ? ORDER BY updated_time DESC",
		appID, t)
	if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if statuses == nil {
		// ensure empty array
		statuses = []app.Status{}
	} else {
		// mark offline instances
		t := time.Now().Add(time.Minute * -5).Unix()
		for i := range statuses {
			if statuses[i].UpdatedTime < t {
				statuses[i].Status = int(client.StatusOffline)
			}
		}
	}

	succeed(c, gin.H{
		"app_id":    appID,
		"instances": statuses,
	})
}
