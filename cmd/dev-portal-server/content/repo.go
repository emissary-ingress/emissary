package content

import (
	"net/url"
	"time"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

type CheckoutOptions struct {
	RepoURL         *url.URL
	RefreshInterval time.Duration
}

type Checkout struct {
	Options CheckoutOptions
	workdir Globbable
	store   storage.Storer
	repo    *git.Repository
}

func NewRepoCheckout(options CheckoutOptions) (checkout *Checkout, err error) {
	checkout = &Checkout{
		workdir: &BetterFS{memfs.New()},
		store:   memory.NewStorage(),
		Options: options,
	}
	err = checkout.clone()
	return
}

func (checkout *Checkout) Fs() Globbable {
	return checkout.workdir
}

func (checkout *Checkout) clone() (err error) {
	checkout.repo, err = git.Clone(checkout.store, checkout.workdir, &git.CloneOptions{
		URL:          checkout.Options.RepoURL.String(),
		SingleBranch: true,
		NoCheckout:   false,
		Depth:        1,
	})
	if err != nil {
		return
	}
	return
}

func (checkout *Checkout) Refresh() (updated bool, err error) {
	worktree, err := checkout.repo.Worktree()
	if err != nil {
		return
	}
	err = worktree.Pull(&git.PullOptions{
		SingleBranch: true,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			err = nil
			return
		}
	}
	updated = true
	return
}
