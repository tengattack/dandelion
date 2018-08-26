package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"../../app"
	"../../log"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/glob"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	// TableName the app configs table
	TableName = "dandelion_app_configs"
	// ParamsError the http bad request for error params
	ParamsError = "Params error"
)

var (
	l *sync.Mutex
)

// InitHandlers init http server handlers
func InitHandlers() error {
	l = new(sync.Mutex)
	return nil
}

func abortWithError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{
		"code": code,
		"info": message,
	})
}

func succeed(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"info": message,
	})
}

func getAppIDs() ([]string, error) {
	// list entries
	branches, err := Repo.Branches()
	if err != nil {
		log.LogError.Errorf("get refs error: %v", err)
		return nil, err
	}
	defer branches.Close()

	var appIDs []string
	var b *plumbing.Reference
	for b, err = branches.Next(); err == nil && b != nil; b, err = branches.Next() {
		if b.Name().IsBranch() {
			appIDs = append(appIDs, b.Name().Short())
		}
	}
	return appIDs, nil
}

func appSyncHandler(c *gin.Context) {
	appID := c.Param("app_id")

	l.Lock()
	defer l.Unlock()

	var err error
	if appID != "" {
		err = Repo.Pull(appID)
		if err != nil && err != git.NoErrAlreadyUpToDate {
			log.LogError.Errorf("pull error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		err = Repo.SyncBranches()
		if err != nil {
			log.LogError.Errorf("sync branches error: %v", err)
			abortWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h, err := Repo.Head()
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
			"app_id":    h.Name().Short(),
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
	commit, err := Repo.CommitObject(plumbing.NewHash(commitID))
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
	config := app.AppConfig{
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

	r, err := DB.NamedExec("INSERT INTO "+TableName+
		" (app_id, status, version, host, instance_id, commit_id, md5sum, author, created_time, updated_time)"+
		" VALUES (:app_id, :status, :version, :host, :instance_id, :commit_id, :md5sum, :author, :created_time, :updated_time)", &config)
	if err != nil {
		log.LogError.Errorf("db insert error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	config.ID, err = r.LastInsertId()
	if err != nil {
		log.LogError.Errorf("get last insert id error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	m := app.NotifyMessage{
		AppID:  appID,
		Event:  "publish",
		Config: config,
	}
	message, err := json.Marshal(m)
	if err != nil {
		log.LogError.Errorf("encode message error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	err = MQ.Publish(string(message))
	if err != nil {
		log.LogError.Errorf("publish message error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	succeed(c, gin.H{
		"app_id": appID,
		"commit": gin.H{
			"commit_id": commit.ID().String(),
			"message":   commit.Message,
			"author": gin.H{
				"name":  commit.Author.Name,
				"email": commit.Author.Email,
				"when":  commit.Author.When,
			},
		},
		"config": config,
	})
}

func appRollbackConfigHandler(c *gin.Context) {
	appID := c.Param("app_id")

	id, _ := strconv.ParseInt(c.PostForm("id"), 10, 64)

	if id <= 0 {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}

	var config app.AppConfig
	err := DB.Get(&config, "SELECT * FROM "+TableName+" WHERE id = ? AND status = 1", id)
	if err == sql.ErrNoRows {
		abortWithError(c, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if config.AppID != appID {
		abortWithError(c, http.StatusForbidden, "config id does not belong to specified app id")
		return
	}

	t := time.Now().Unix()
	_, err = DB.Exec("UPDATE "+TableName+" SET status = 0, updated_time = ? WHERE id = ?", t, id)
	if err != nil {
		log.LogError.Errorf("db update error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// rollback, notify all nodes
	m := app.NotifyMessage{
		AppID:  appID,
		Event:  "rollback",
		Config: config,
	}
	message, err := json.Marshal(m)
	if err != nil {
		log.LogError.Errorf("encode message error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	err = MQ.Publish(string(message))
	if err != nil {
		log.LogError.Errorf("publish message error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	succeed(c, gin.H{
		"app_id": appID,
		"config": config,
	})
}

func appMatchConfigHandler(c *gin.Context) {
	appID := c.Param("app_id")

	version := c.Query("version")
	host := c.Query("host")
	instanceID := c.Query("instance_id")

	var configs []app.AppConfig
	// TODO: apply limit & offset
	err := DB.Select(&configs, "SELECT * FROM "+TableName+" WHERE app_id = ? AND status = 1 AND version <= ? ORDER BY created_time DESC",
		appID, version)
	if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for _, config := range configs {
		glob1, err1 := glob.Compile(config.Host)
		glob2, err2 := glob.Compile(config.InstanceID)
		if err1 != nil || err2 != nil {
			log.LogAccess.Warnf("config %d host or instance_id glob complie failed", config.ID)
			continue
		}
		if glob1.Match(host) && glob2.Match(instanceID) {
			succeed(c, gin.H{
				"app_id": appID,
				"config": config,
			})
			return
		}
	}

	abortWithError(c, http.StatusNotFound, "not found matched config")
}

func appListConfigsHandler(c *gin.Context) {
	appID := c.Param("app_id")

	var configs []app.AppConfig
	err := DB.Select(&configs, "SELECT * FROM "+TableName+" WHERE app_id = ? AND status = 1 ORDER BY created_time DESC",
		appID)
	if err != nil {
		log.LogError.Errorf("db select error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if configs == nil {
		// empty array
		configs = make([]app.AppConfig, 0)
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

	commits, err := Repo.Log(&git.LogOptions{})
	if err != nil {
		log.LogError.Errorf("log error: %v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer commits.Close()
	var r []gin.H
	for commit, err := commits.Next(); err == nil && commit != nil; commit, err = commits.Next() {
		r = append(r, gin.H{
			"commit_id": commit.ID().String(),
			"message":   commit.Message,
			"author": gin.H{
				"name":  commit.Author.Name,
				"email": commit.Author.Email,
				"when":  commit.Author.When,
			},
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

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := Repo.CommitObject(plumbing.NewHash(commitID))
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
	appID := c.Param("app_id")
	commitID := c.Param("commit_id")
	path := c.Param("path")

	// remove the beginning slash
	path = path[1:]

	l.Lock()
	defer l.Unlock()

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

	// TODO: check commit id belongs to this app id

	// ... retrieving the commit object
	commit, err := Repo.CommitObject(plumbing.NewHash(commitID))
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
