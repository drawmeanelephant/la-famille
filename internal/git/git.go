package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Repo binds a set of git commands to a repository working directory. The
// zero value (empty Dir) keeps the historical behavior of running in the
// process's current working directory; commands like `pr sync --project-root
// <dir>` bind a Repo to the flag's target so the CWD repository can never
// silently win (#640).
type Repo struct {
	// Dir is passed through to exec.Cmd.Dir; empty means the process working
	// directory.
	Dir string
}

func (r Repo) command(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	return cmd
}

// HasUncommittedChanges returns true if there are uncommitted changes in the working directory.
func (r Repo) HasUncommittedChanges() (bool, error) {
	cmd := r.command("status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return strings.TrimSpace(out.String()) != "", nil
}

// GetRemoteURL returns the URL of the specified remote (usually "origin").
func (r Repo) GetRemoteURL(remote string) (string, error) {
	cmd := r.command("remote", "get-url", remote)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get remote url for %s: %w", remote, err)
	}
	return strings.TrimSpace(out.String()), nil
}

// ParseOwnerRepo extracts the owner and repository name from a git remote URL.
// It handles both HTTPS and SSH formats.
func ParseOwnerRepo(url string) (string, string, error) {
	// Examples:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// https://github.com/owner/repo

	url = strings.TrimSuffix(url, ".git")

	var pathPart string
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// http(s)://github.com/owner/repo
		parts := strings.SplitN(url, "github.com/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("could not parse HTTPS github URL: %s", url)
		}
		pathPart = parts[1]
	} else if strings.HasPrefix(url, "git@") {
		// git@github.com:owner/repo
		parts := strings.SplitN(url, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("could not parse SSH github URL: %s", url)
		}
		pathPart = parts[1]
	} else {
		return "", "", fmt.Errorf("unsupported git remote URL format: %s", url)
	}

	parts := strings.SplitN(pathPart, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("could not extract owner/repo from path: %s", pathPart)
	}

	return parts[0], parts[1], nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func (r Repo) CurrentBranch() (string, error) {
	cmd := r.command("rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %s: %w", stderr.String(), err)
	}
	branch := strings.TrimSpace(out.String())
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("not on a named branch (detached HEAD)")
	}
	return branch, nil
}

// CheckoutBranch creates and checks out a new branch.
func (r Repo) CheckoutBranch(branchName string) error {
	cmd := r.command("checkout", "-b", branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %s: %w", branchName, stderr.String(), err)
	}
	return nil
}

// Checkout switches to an existing branch.
func (r Repo) Checkout(branchName string) error {
	cmd := r.command("checkout", branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %s: %w", branchName, stderr.String(), err)
	}
	return nil
}

// AddAll stages all changes.
func (r Repo) AddAll() error {
	cmd := r.command("add", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to git add: %s: %w", stderr.String(), err)
	}
	return nil
}

// Commit creates a commit with the specified message and author.
func (r Repo) Commit(message string, authorName string, authorEmail string) error {
	author := fmt.Sprintf("%s <%s>", authorName, authorEmail)
	cmd := r.command("commit", "-m", message, "--author", author)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %s: %w", stderr.String(), err)
	}
	return nil
}

// Push pushes the specified branch to the remote.
func (r Repo) Push(remote string, branchName string) error {
	// Set upstream so that the branch tracks correctly.
	cmd := r.command("push", "--set-upstream", remote, branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %s: %w", branchName, remote, stderr.String(), err)
	}
	return nil
}
