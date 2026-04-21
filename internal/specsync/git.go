package specsync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ensureRepo(workDir, repoURL, tag string) error {
	gitDir := filepath.Join(workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(workDir), 0o755); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "=> clone %s\n", repoURL)
		cmd := exec.Command("git", "clone", "--filter=blob:none", "--quiet", repoURL, workDir)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		fmt.Fprintf(os.Stderr, "=> fetch %s\n", workDir)
		cmd := exec.Command("git", "-C", workDir, "fetch", "--tags", "--quiet")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git fetch: %w", err)
		}
	}
	cmd := exec.Command("git", "-C", workDir, "-c", "advice.detachedHead=false", "checkout", "--quiet", tag)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s: %w", tag, err)
	}
	return nil
}
