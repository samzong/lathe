package specsync

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ensureRepo clones or updates workDir to the given tag and returns the
// resolved commit SHA. A tag moved upstream would produce a different SHA on
// the next sync; VerifyState compares the stored sha to enforce that.
func ensureRepo(workDir, repoURL, tag string) (string, error) {
	gitDir := filepath.Join(workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(workDir), 0o755); err != nil {
			return "", err
		}
		fmt.Fprintf(os.Stderr, "=> clone %s\n", repoURL)
		cmd := exec.Command("git", "clone", "--filter=blob:none", "--quiet", repoURL, workDir)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git clone: %w", err)
		}
	} else if err != nil {
		return "", err
	} else {
		fmt.Fprintf(os.Stderr, "=> fetch %s\n", workDir)
		cmd := exec.Command("git", "-C", workDir, "fetch", "--tags", "--quiet")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git fetch: %w", err)
		}
	}
	checkout := exec.Command("git", "-C", workDir, "-c", "advice.detachedHead=false", "checkout", "--quiet", tag)
	checkout.Stderr = os.Stderr
	if err := checkout.Run(); err != nil {
		return "", fmt.Errorf("git checkout %s: %w", tag, err)
	}

	var out bytes.Buffer
	rev := exec.Command("git", "-C", workDir, "rev-parse", "HEAD")
	rev.Stdout = &out
	rev.Stderr = os.Stderr
	if err := rev.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	sha := string(bytes.TrimSpace(out.Bytes()))
	if len(sha) != 40 || !isHex(sha) {
		return "", fmt.Errorf("unexpected rev-parse output %q", sha)
	}
	return sha, nil
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}
