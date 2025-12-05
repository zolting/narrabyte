package tools

// Repository identifies which repository context a path belongs to.
type Repository string

const (
	// RepositoryDocs refers to the documentation repository (temp workspace).
	RepositoryDocs Repository = "docs"
	// RepositoryCode refers to the source codebase repository.
	RepositoryCode Repository = "code"
)

// IsValid checks if the repository value is valid.
func (r Repository) IsValid() bool {
	return r == RepositoryDocs || r == RepositoryCode
}

// String returns the string representation of the repository.
func (r Repository) String() string {
	return string(r)
}

// IsDocsOnly returns true if the repository only allows docs operations.
func (r Repository) IsDocsOnly() bool {
	return r == RepositoryDocs
}
