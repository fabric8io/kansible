package vcs

import (
	"io/ioutil"
	"time"
	//"log"
	"os"
	"testing"
)

// Canary test to ensure GitRepo implements the Repo interface.
var _ Repo = &GitRepo{}

// To verify git is working we perform integration testing
// with a known git service.

func TestGit(t *testing.T) {

	tempDir, err := ioutil.TempDir("", "go-vcs-git-tests")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
	}()

	repo, err := NewGitRepo("https://github.com/Masterminds/VCSTestRepo", tempDir+"/VCSTestRepo")
	if err != nil {
		t.Error(err)
	}

	if repo.Vcs() != Git {
		t.Error("Git is detecting the wrong type")
	}

	// Check the basic getters.
	if repo.Remote() != "https://github.com/Masterminds/VCSTestRepo" {
		t.Error("Remote not set properly")
	}
	if repo.LocalPath() != tempDir+"/VCSTestRepo" {
		t.Error("Local disk location not set properly")
	}

	//Logger = log.New(os.Stdout, "", log.LstdFlags)

	// Do an initial clone.
	err = repo.Get()
	if err != nil {
		t.Errorf("Unable to clone Git repo. Err was %s", err)
	}

	// Verify Git repo is a Git repo
	if repo.CheckLocal() == false {
		t.Error("Problem checking out repo or Git CheckLocal is not working")
	}

	// Test internal lookup mechanism used outside of Git specific functionality.
	ltype, err := DetectVcsFromFS(tempDir + "/VCSTestRepo")
	if err != nil {
		t.Error("detectVcsFromFS unable to Git repo")
	}
	if ltype != Git {
		t.Errorf("detectVcsFromFS detected %s instead of Git type", ltype)
	}

	// Test NewRepo on existing checkout. This should simply provide a working
	// instance without error based on looking at the local directory.
	nrepo, nrerr := NewRepo("https://github.com/Masterminds/VCSTestRepo", tempDir+"/VCSTestRepo")
	if nrerr != nil {
		t.Error(nrerr)
	}
	// Verify the right oject is returned. It will check the local repo type.
	if nrepo.CheckLocal() == false {
		t.Error("Wrong version returned from NewRepo")
	}

	// Perform an update.
	err = repo.Update()
	if err != nil {
		t.Error(err)
	}

	// Set the version using the short hash.
	err = repo.UpdateVersion("806b07b")
	if err != nil {
		t.Errorf("Unable to update Git repo version. Err was %s", err)
	}

	// Once a ref has been checked out the repo is in a detached head state.
	// Trying to pull in an update in this state will cause an error. Update
	// should cleanly handle this. Pulling on a branch (tested elsewhere) and
	// skipping that here.
	err = repo.Update()
	if err != nil {
		t.Error(err)
	}

	// Use Version to verify we are on the right version.
	v, err := repo.Version()
	if v != "806b07b08faa21cfbdae93027904f80174679402" {
		t.Error("Error checking checked out Git version")
	}
	if err != nil {
		t.Error(err)
	}

	// Use Date to verify we are on the right commit.
	d, err := repo.Date()
	if d.Format(longForm) != "2015-07-29 09:46:39 -0400" {
		t.Error("Error checking checked out Git commit date")
	}
	if err != nil {
		t.Error(err)
	}

	// Verify that we can set the version something other than short hash
	err = repo.UpdateVersion("master")
	if err != nil {
		t.Errorf("Unable to update Git repo version. Err was %s", err)
	}
	err = repo.UpdateVersion("806b07b08faa21cfbdae93027904f80174679402")
	if err != nil {
		t.Errorf("Unable to update Git repo version. Err was %s", err)
	}
	v, err = repo.Version()
	if v != "806b07b08faa21cfbdae93027904f80174679402" {
		t.Error("Error checking checked out Git version")
	}
	if err != nil {
		t.Error(err)
	}

	tags, err := repo.Tags()
	if err != nil {
		t.Error(err)
	}
	if tags[0] != "1.0.0" {
		t.Error("Git tags is not reporting the correct version")
	}

	branches, err := repo.Branches()
	if err != nil {
		t.Error(err)
	}
	// The branches should be HEAD, master, and test.
	if branches[2] != "test" {
		t.Error("Git is incorrectly returning branches")
	}

	if repo.IsReference("1.0.0") != true {
		t.Error("Git is reporting a reference is not one")
	}

	if repo.IsReference("foo") == true {
		t.Error("Git is reporting a non-existant reference is one")
	}

	if repo.IsDirty() == true {
		t.Error("Git incorrectly reporting dirty")
	}

	ci, err := repo.CommitInfo("806b07b08faa21cfbdae93027904f80174679402")
	if err != nil {
		t.Error(err)
	}
	if ci.Commit != "806b07b08faa21cfbdae93027904f80174679402" {
		t.Error("Git.CommitInfo wrong commit id")
	}
	if ci.Author != "Matt Farina <matt@mattfarina.com>" {
		t.Error("Git.CommitInfo wrong author")
	}
	if ci.Message != "Update README.md" {
		t.Error("Git.CommitInfo wrong message")
	}
	ti, err := time.Parse(time.RFC1123Z, "Wed, 29 Jul 2015 09:46:39 -0400")
	if err != nil {
		t.Error(err)
	}
	if !ti.Equal(ci.Date) {
		t.Error("Git.CommitInfo wrong date")
	}

	_, err = repo.CommitInfo("asdfasdfasdf")
	if err != ErrRevisionUnavailable {
		t.Error("Git didn't return expected ErrRevisionUnavailable")
	}
}

func TestGitCheckLocal(t *testing.T) {
	// Verify repo.CheckLocal fails for non-Git directories.
	// TestGit is already checking on a valid repo
	tempDir, err := ioutil.TempDir("", "go-vcs-git-tests")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
	}()

	repo, _ := NewGitRepo("", tempDir)
	if repo.CheckLocal() == true {
		t.Error("Git CheckLocal does not identify non-Git location")
	}

	// Test NewRepo when there's no local. This should simply provide a working
	// instance without error based on looking at the remote localtion.
	_, nrerr := NewRepo("https://github.com/Masterminds/VCSTestRepo", tempDir+"/VCSTestRepo")
	if nrerr != nil {
		t.Error(nrerr)
	}
}
