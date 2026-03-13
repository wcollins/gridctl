package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/gridctl/gridctl/pkg/builder"
	"github.com/gridctl/gridctl/pkg/registry"
)

// CloneResult contains the result of a clone + discovery operation.
type CloneResult struct {
	RepoPath  string
	CommitSHA string
	Skills    []DiscoveredSkill
}

// DiscoveredSkill represents a SKILL.md found in a cloned repo.
type DiscoveredSkill struct {
	Name        string
	Path        string // Relative path from repo root to SKILL.md directory
	Skill       *registry.AgentSkill
	ContentHash string
}

// CloneAndDiscover clones a repo and discovers all SKILL.md files.
func CloneAndDiscover(repo, ref, subPath string, logger *slog.Logger) (*CloneResult, error) {
	repoPath, err := cloneShallow(repo, ref, logger)
	if err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	commitSHA, err := getHeadCommit(repoPath)
	if err != nil {
		return nil, fmt.Errorf("getting HEAD commit: %w", err)
	}

	searchDir := repoPath
	if subPath != "" {
		searchDir = filepath.Join(repoPath, subPath)
		if _, err := os.Stat(searchDir); err != nil {
			return nil, fmt.Errorf("path %q not found in repository: %w", subPath, err)
		}
	}

	skills, err := discoverSkills(searchDir, repoPath)
	if err != nil {
		return nil, fmt.Errorf("discovering skills: %w", err)
	}

	return &CloneResult{
		RepoPath:  repoPath,
		CommitSHA: commitSHA,
		Skills:    skills,
	}, nil
}

// FetchAndCompare fetches the latest from a remote and compares with current.
func FetchAndCompare(repo, ref string, currentSHA string, logger *slog.Logger) (string, bool, error) {
	repoPath, err := builder.URLToPath(repo)
	if err != nil {
		return "", false, fmt.Errorf("getting cache path: %w", err)
	}

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		// Repo not cached, needs full clone
		return "", true, nil
	}

	fetchOpts := &git.FetchOptions{Force: true}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		fetchOpts.Auth = &http.BasicAuth{Username: token, Password: ""}
	}

	if err := r.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
		logger.Warn("fetch failed", "error", err)
		return currentSHA, false, nil
	}

	if ref != "" {
		newSHA, err := resolveRef(r, ref)
		if err != nil {
			return currentSHA, false, nil
		}
		return newSHA, newSHA != currentSHA, nil
	}

	head, err := r.Head()
	if err != nil {
		return currentSHA, false, nil
	}
	newSHA := head.Hash().String()
	return newSHA, newSHA != currentSHA, nil
}

// ListRemoteTags returns all tags from a remote repository.
func ListRemoteTags(repoPath string) ([]string, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	tagIter, err := r.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var tags []string
	if err := tagIter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, ref.Name().Short())
		return nil
	}); err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tags, nil
}

func cloneShallow(url, ref string, logger *slog.Logger) (string, error) {
	if err := builder.EnsureReposCacheDir(); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	repoPath, err := builder.URLToPath(url)
	if err != nil {
		return "", fmt.Errorf("getting cache path: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:   url,
		Depth: 1,
		Tags:  git.AllTags,
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneOpts.Auth = &http.BasicAuth{Username: token, Password: ""}
	}

	// If repo exists, fetch updates instead
	if _, err := os.Stat(repoPath); err == nil {
		return updateExisting(repoPath, ref, logger)
	}

	logger.Info("cloning repository", "url", url)

	if ref != "" && !IsSemVerConstraint(ref) {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref)
		cloneOpts.SingleBranch = true
	}

	r, err := git.PlainClone(repoPath, false, cloneOpts)
	if err != nil {
		// Retry without single-branch restriction
		if ref != "" {
			_ = os.RemoveAll(repoPath)
			cloneOpts.SingleBranch = false
			cloneOpts.ReferenceName = ""
			r, err = git.PlainClone(repoPath, false, cloneOpts)
		}
		if err != nil {
			return "", fmt.Errorf("cloning: %w", err)
		}
	}

	// Handle semver constraints or specific refs
	if ref != "" {
		if IsSemVerConstraint(ref) {
			tags, err := ListRemoteTags(repoPath)
			if err != nil {
				return "", err
			}
			resolvedTag, err := ResolveSemVerConstraint(ref, tags)
			if err != nil {
				return "", err
			}
			if err := checkoutRef(r, resolvedTag); err != nil {
				return "", err
			}
		} else {
			if err := checkoutRef(r, ref); err != nil {
				return "", err
			}
		}
	}

	return repoPath, nil
}

func updateExisting(repoPath, ref string, logger *slog.Logger) (string, error) {
	logger.Info("updating cached repository")

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		_ = os.RemoveAll(repoPath)
		return "", fmt.Errorf("opening cached repo (removed): %w", err)
	}

	fetchOpts := &git.FetchOptions{Force: true, Tags: git.AllTags}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		fetchOpts.Auth = &http.BasicAuth{Username: token, Password: ""}
	}

	if err := r.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
		logger.Warn("fetch failed, using cached", "error", err)
	}

	if ref != "" {
		if IsSemVerConstraint(ref) {
			tags, err := ListRemoteTags(repoPath)
			if err != nil {
				logger.Warn("failed to list tags, using cached", "error", err)
				return repoPath, nil
			}
			resolvedTag, err := ResolveSemVerConstraint(ref, tags)
			if err != nil {
				logger.Warn("failed to resolve constraint, using cached", "constraint", ref, "error", err)
				return repoPath, nil
			}
			if err := checkoutRef(r, resolvedTag); err != nil {
				logger.Warn("failed to checkout tag, using cached", "tag", resolvedTag, "error", err)
				return repoPath, nil
			}
		} else {
			if err := checkoutRef(r, ref); err != nil {
				logger.Warn("failed to checkout ref, using cached", "ref", ref, "error", err)
			}
		}
	}

	return repoPath, nil
}

func checkoutRef(r *git.Repository, ref string) error {
	wt, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	// Try tag first
	tagRef := plumbing.NewTagReferenceName(ref)
	if err := wt.Checkout(&git.CheckoutOptions{Branch: tagRef, Force: true}); err == nil {
		return nil
	}

	// Try branch
	branchRef := plumbing.NewBranchReferenceName(ref)
	if err := wt.Checkout(&git.CheckoutOptions{Branch: branchRef, Force: true}); err == nil {
		return nil
	}

	// Try remote branch
	remoteRef := plumbing.NewRemoteReferenceName("origin", ref)
	if err := wt.Checkout(&git.CheckoutOptions{Branch: remoteRef, Force: true}); err == nil {
		return nil
	}

	// Try commit hash
	hash := plumbing.NewHash(ref)
	if !hash.IsZero() {
		if err := wt.Checkout(&git.CheckoutOptions{Hash: hash, Force: true}); err == nil {
			return nil
		}
	}

	return fmt.Errorf("unable to checkout ref %q", ref)
}

func resolveRef(r *git.Repository, ref string) (string, error) {
	// Try tag
	tagRef, err := r.Tag(ref)
	if err == nil {
		return tagRef.Hash().String(), nil
	}

	// Try remote branch
	remoteRef, err := r.Reference(plumbing.NewRemoteReferenceName("origin", ref), true)
	if err == nil {
		return remoteRef.Hash().String(), nil
	}

	// Try branch
	branchRef, err := r.Reference(plumbing.NewBranchReferenceName(ref), true)
	if err == nil {
		return branchRef.Hash().String(), nil
	}

	return "", fmt.Errorf("unable to resolve ref %q", ref)
}

func getHeadCommit(repoPath string) (string, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}
	head, err := r.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
}

func discoverSkills(searchDir, repoRoot string) ([]DiscoveredSkill, error) {
	var skills []DiscoveredSkill

	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		sk, err := registry.ParseSkillMD(data)
		if err != nil {
			return nil
		}

		skillDir := filepath.Dir(path)
		relPath, _ := filepath.Rel(repoRoot, skillDir)
		dirName := filepath.Base(skillDir)

		if sk.Name == "" {
			sk.Name = dirName
		}

		skills = append(skills, DiscoveredSkill{
			Name:        sk.Name,
			Path:        relPath,
			Skill:       sk,
			ContentHash: contentHash(data),
		})

		return nil
	})

	return skills, err
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ContentHashFile computes a SHA-256 hash of a file.
func ContentHashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return contentHash(data), nil
}

// SafeRepoPath validates a path component to prevent directory traversal.
func SafeRepoPath(path string) error {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("invalid path: %q", path)
	}
	return nil
}
