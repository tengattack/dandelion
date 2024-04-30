package repository

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/tengattack/tgo/logger"
)

// Config for repository
type Config struct {
	RepositoryPath string `yaml:"repository_path"`
	RemoteURL      string `yaml:"remote_url"`
	HTTPProxy      string `yaml:"http_proxy"`
}

// Repository the repository structure
type Repository struct {
	RepositoryPath string
	Repo           *git.Repository
}

var (
	pullOptions  *git.PullOptions
	pushOptions  *git.PushOptions
	fetchOptions *git.FetchOptions
)

// isDirExists check dir exists
func isDirExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

// InitRepository init or open the repository
func InitRepository(repoConf *Config) (*Repository, error) {
	repoPath := repoConf.RepositoryPath
	remoteURL := repoConf.RemoteURL
	httpProxy := repoConf.HTTPProxy

	var auth transport.AuthMethod
	if strings.HasPrefix(remoteURL, "git@") {
		// Git SSH
		homePath := os.Getenv("HOME")
		keyPath := path.Join(homePath, ".ssh/id_rsa")
		logger.Debugf("ssh key path: %s", keyPath)

		sshKey, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey([]byte(sshKey))
		if err != nil {
			return nil, err
		}
		auth = &gitssh.PublicKeys{
			User:   "git",
			Signer: signer,
			HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
				HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
					logger.Infof("git auth, host: %s (%s) pubkey: %s:%x", hostname, remote, key.Type(), key.Marshal())
					return nil
				},
			},
		}
	} else if strings.HasPrefix(remoteURL, "http:") || strings.HasPrefix(remoteURL, "https:") {
		// Git HTTP
		u, err := url.Parse(remoteURL)
		if err != nil {
			return nil, err
		}
		if u.User != nil {
			pass, _ := u.User.Password()
			if u.User.Username() != "" && pass != "" {
				auth = &githttp.BasicAuth{
					Username: u.User.Username(),
					Password: pass,
				}
			} else if u.User.Username() != "" {
				auth = &githttp.TokenAuth{
					Token: u.User.Username(),
				}
			}
		}
		u.User = nil
		remoteURL = u.String()

		if httpProxy != "" {
			proxyURL, err := url.Parse(httpProxy)
			if err != nil {
				return nil, err
			}
			customClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 60 * time.Second, // 60 seconds timeout
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // don't follow redirect
				},
			}
			// Override http(s) default protocol to use our custom client
			gitclient.InstallProtocol("http", githttp.NewClient(customClient))
			gitclient.InstallProtocol("https", githttp.NewClient(customClient))
		}
	}
	pullOptions = &git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
		Force:      true,
	}
	pushOptions = &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
	}
	fetchOptions = &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
		Force:      true,
	}

	var repo *git.Repository
	var err error
	newRepo := false
	if !isDirExists(repoPath) {
		repo, err = git.PlainInit(repoPath, false)
		newRepo = true
	} else {
		repo, err = git.PlainOpen(repoPath)
	}
	if err != nil {
		logger.Errorf("init/open repository error: %v", err)
		return nil, err
	}

	// TODO: update remote if not same
	_, err = repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	if err != nil && err != git.ErrRemoteExists {
		logger.Errorf("create remote error: %v", err)
		return nil, err
	}

	r := &Repository{
		RepositoryPath: repoPath,
		Repo:           repo,
	}

	if newRepo {
		logger.Info("fetching new repo...")
		err = r.SyncBranches()
		if err != nil {
			logger.Errorf("sync branches error: %v", err)
			return nil, err
		}
	}

	return r, nil
}

// Head returns the reference where HEAD is pointing to.
func (r *Repository) Head() (*plumbing.Reference, error) {
	return r.Repo.Head()
}

// Fetch branches and/or tags
func (r *Repository) Fetch() error {
	return r.Repo.Fetch(fetchOptions)
}

// Pull incorporates changes from a remote repository into the specified branch
func (r *Repository) Pull(branch string) error {
	wt, err := r.Repo.Worktree()
	if err != nil {
		logger.Errorf("worktree error: %v", err)
		return err
	}
	po := pullOptions
	po.ReferenceName = plumbing.ReferenceName("refs/heads/" + branch)
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Force:  true,
	})
	if err != nil {
		logger.Errorf("checkout error: %v", err)
		return err
	}
	return wt.Pull(po)
}

// Log returns the commit history from the given LogOptions.
func (r *Repository) Log(o *git.LogOptions) (object.CommitIter, error) {
	return r.Repo.Log(o)
}

// CommitObject return a Commit with the given hash. If not found
// plumbing.ErrObjectNotFound is returned.
func (r *Repository) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	return r.Repo.CommitObject(h)
}

// Branches returns all the References that are Branches.
func (r *Repository) Branches() (storer.ReferenceIter, error) {
	return r.Repo.Branches()
}

// Worktree returns a worktree based on the given fs, if nil the default
// worktree will be used.
func (r *Repository) Worktree() (*git.Worktree, error) {
	return r.Repo.Worktree()
}

// SyncBranches sync branches from remote
func (r *Repository) SyncBranches() error {
	err := r.Fetch()
	if err == git.NoErrAlreadyUpToDate {
		// already up to update
		logger.Debugf("sync branches: %v", err)
	} else if err != nil {
		return err
	}

	refs, err := r.Repo.References()
	if err != nil {
		logger.Errorf("get remotes error: %v", err)
		return err
	}
	defer refs.Close()

	var ref *plumbing.Reference
	for ref, err = refs.Next(); err == nil && ref != nil; ref, err = refs.Next() {
		if ref.Name().IsRemote() {
			branchName := strings.SplitN(ref.Name().Short(), "/", 2)[1]

			logger.Debugf("ref: %s -> %s", branchName, ref.Name())
			branchRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+branchName), ref.Hash())
			// The created reference is saved in the storage.
			err = r.Repo.Storer.SetReference(branchRef)

			/*err = r.Repo.CreateBranch(&gitconfig.Branch{
				Name:   branchName,
				Remote: "origin",
				Merge:  plumbing.ReferenceName("refs/heads/" + branchName),
			})*/
			if err != nil {
				return err
			}
		}
	}
	return nil
}
