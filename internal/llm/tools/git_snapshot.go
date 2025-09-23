package tools

import (
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	ErrSnapshotEscapes   = errors.New("path escapes repository root")
	ErrSnapshotDirectory = errors.New("path refers to a directory")
	ErrSnapshotNotFound  = errors.New("path not found in snapshot")
	errListLimitReached  = errors.New("list limit reached")
)

type GitSnapshot struct {
	repo   *git.Repository
	commit *object.Commit
	tree   *object.Tree
	root   string
	hash   plumbing.Hash
	branch string
}

type GitTreeEntry struct {
	Name string
	Path string
	Mode filemode.FileMode
}

func (e GitTreeEntry) IsDir() bool {
	return e.Mode == filemode.Dir
}

func (e GitTreeEntry) IsFile() bool {
	switch e.Mode {
	case filemode.Regular, filemode.Executable, filemode.Symlink:
		return true
	default:
		return false
	}
}

func NewGitSnapshot(repo *git.Repository, commit *object.Commit, root, branch string) (*GitSnapshot, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if commit == nil {
		return nil, fmt.Errorf("commit is required")
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	cleaned := strings.TrimSpace(root)
	if cleaned != "" {
		if abs, err := filepath.Abs(cleaned); err == nil {
			cleaned = abs
		}
	}
	return &GitSnapshot{
		repo:   repo,
		commit: commit,
		tree:   tree,
		root:   cleaned,
		hash:   commit.Hash,
		branch: strings.TrimSpace(branch),
	}, nil
}

func (s *GitSnapshot) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *GitSnapshot) CommitHash() plumbing.Hash {
	if s == nil {
		return plumbing.Hash{}
	}
	return s.hash
}

func (s *GitSnapshot) Branch() string {
	if s == nil {
		return ""
	}
	return s.branch
}

func (s *GitSnapshot) relativeFromAbs(absPath string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("git snapshot not configured")
	}
	candidate := absPath
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.root, candidate)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	absRoot := s.root
	if absRoot == "" {
		return "", ErrSnapshotEscapes
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return ".", nil
	}
	if strings.HasPrefix(rel, "..") {
		return "", ErrSnapshotEscapes
	}
	return filepath.ToSlash(rel), nil
}

func (s *GitSnapshot) treeFor(rel string) (*object.Tree, error) {
	if s == nil {
		return nil, fmt.Errorf("git snapshot not configured")
	}
	cleaned := strings.TrimSpace(rel)
	if cleaned == "" || cleaned == "." {
		return s.tree, nil
	}
	cleaned = path.Clean(cleaned)
	subtree, err := s.tree.Tree(cleaned)
	if err != nil {
		if errors.Is(err, object.ErrDirectoryNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, err
	}
	return subtree, nil
}

func (s *GitSnapshot) list(rel string) ([]GitTreeEntry, error) {
	tree, err := s.treeFor(rel)
	if err != nil {
		return nil, err
	}
	entries := make([]GitTreeEntry, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		entries = append(entries, GitTreeEntry{
			Name: entry.Name,
			Path: joinCommitPath(rel, entry.Name),
			Mode: entry.Mode,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s *GitSnapshot) readFile(rel string) ([]byte, bool, error) {
	if s == nil {
		return nil, false, fmt.Errorf("git snapshot not configured")
	}
	cleaned := strings.TrimSpace(rel)
	if cleaned == "" || cleaned == "." {
		return nil, false, ErrSnapshotDirectory
	}
	if subtree, err := s.treeFor(cleaned); err == nil && subtree != nil {
		return nil, false, ErrSnapshotDirectory
	}
	file, err := s.commit.File(path.Clean(cleaned))
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return nil, false, ErrSnapshotNotFound
		}
		return nil, false, err
	}
	isBinary, err := file.IsBinary()
	if err != nil {
		return nil, false, err
	}
	reader, err := file.Reader()
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}
	return data, isBinary, nil
}

func (s *GitSnapshot) suggestions(dirRel, baseName string, limit int) []string {
	tree, err := s.treeFor(dirRel)
	if err != nil {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(baseName))
	if needle == "" {
		return nil
	}
	type candidate struct {
		name  string
		score int
	}
	var results []candidate
	for _, entry := range tree.Entries {
		name := entry.Name
		lower := strings.ToLower(name)
		if strings.Contains(lower, needle) || strings.Contains(needle, lower) {
			results = append(results, candidate{name: name, score: len(name)})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].name < results[j].name
		}
		return results[i].score < results[j].score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.name)
	}
	return out
}

func joinCommitPath(base, name string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return name
	}
	return path.Join(base, name)
}

func (s *GitSnapshot) walkFiles(rel string, fn func(relPath string, entry GitTreeEntry, file *object.File) error) error {
	entries, err := s.list(rel)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := s.walkFiles(entry.Path, fn); err != nil {
				if errors.Is(err, errListLimitReached) {
					return err
				}
				return err
			}
			continue
		}
		file, err := s.commit.File(entry.Path)
		if err != nil {
			if errors.Is(err, object.ErrFileNotFound) {
				continue
			}
			return err
		}
		if err := fn(entry.Path, entry, file); err != nil {
			if errors.Is(err, errListLimitReached) {
				return err
			}
			return err
		}
	}
	return nil
}
