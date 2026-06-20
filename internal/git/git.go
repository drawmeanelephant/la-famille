package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// HasUncommittedChanges returns true if there are uncommitted changes in the working directory.
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return strings.TrimSpace(out.String()) != "", nil
}

// GetRemoteURL returns the URL of the specified remote (usually "origin").
func GetRemoteURL(remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
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

// CheckoutBranch creates and checks out a new branch.
func CheckoutBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %s: %w", branchName, stderr.String(), err)
	}
	return nil
}

// AddAll stages all changes.
func AddAll() error {
	cmd := exec.Command("git", "add", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to git add: %s: %w", stderr.String(), err)
	}
	return nil
}

// Commit creates a commit with the specified message and author.
func Commit(message string, authorName string, authorEmail string) error {
	author := fmt.Sprintf("%s <%s>", authorName, authorEmail)
	cmd := exec.Command("git", "commit", "-m", message, "--author", author)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %s: %w", stderr.String(), err)
	}
	return nil
}

// Push pushes the specified branch to the remote.
func Push(remote string, branchName string) error {
	// Set upstream so that the branch tracks correctly.
	cmd := exec.Command("git", "push", "--set-upstream", remote, branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %s: %w", branchName, remote, stderr.String(), err)
	}
	return nil
}
