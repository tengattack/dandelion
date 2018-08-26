package repository

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"

	"github.com/tengattack/dandelion/log"

	"golang.org/x/crypto/ssh"
	git "gopkg.in/src-d/go-git.v4"
	gitconfig "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

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
func InitRepository(repoPath, remoteURL string) (*Repository, error) {
	homePath := os.Getenv("HOME")
	keyPath := path.Join(homePath, ".ssh/id_rsa")
	log.LogAccess.Debugf("ssh key path: %s", keyPath)

	sshKey, err := ioutil.ReadFile(keyPath)
	signer, err := ssh.ParsePrivateKey([]byte(sshKey))
	auth := &gitssh.PublicKeys{
		User:   "git",
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				log.LogAccess.Infof("git auth, host: %s (%s) pubkey: %s:%x", hostname, remote, key.Type(), key.Marshal())
				return nil
			},
		},
	}
	if err != nil {
		return nil, err
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
	newRepo := false
	if !isDirExists(repoPath) {
		repo, err = git.PlainInit(repoPath, false)
		newRepo = true
	} else {
		repo, err = git.PlainOpen(repoPath)
	}
	if err != nil {
		log.LogError.Errorf("init/open repository error: %v", err)
		return nil, err
	}

	// TODO: update remote if not same
	_, err = repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	if err != nil && err != git.ErrRemoteExists {
		log.LogError.Errorf("create remote error: %v", err)
		return nil, err
	}

	r := &Repository{
		RepositoryPath: repoPath,
		Repo:           repo,
	}

	if newRepo {
		log.LogAccess.Info("fetching new repo...")
		err = r.SyncBranches()
		if err != nil {
			log.LogError.Errorf("sync branches error: %v", err)
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
		log.LogError.Errorf("worktree error: %v", err)
		return err
	}
	po := pullOptions
	po.ReferenceName = plumbing.ReferenceName("refs/heads/" + branch)
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Force:  true,
	})
	if err != nil {
		log.LogError.Errorf("checkout error: %v", err)
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
		log.LogAccess.Debugf("sync branches: %v", err)
		return nil
	} else if err != nil {
		return err
	}
	refs, err := r.Repo.References()
	if err != nil {
		log.LogError.Errorf("get remotes error: %v", err)
		return err
	}
	defer refs.Close()

	var ref *plumbing.Reference
	for ref, err = refs.Next(); err == nil && ref != nil; ref, err = refs.Next() {
		if ref.Name().IsRemote() {
			branchName := strings.Split(ref.Name().Short(), "/")[1]

			log.LogAccess.Debugf("ref: %s -> %s", branchName, ref.Name())
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
