package tools

import (
	"context"
	"log"
	"narrabyte/internal/services"
	"strings"
)

type FilePermissionsUtil struct {
	RepoLinks services.RepoLinkService
}

func (f *FilePermissionsUtil) CheckReadPermissions(context context.Context, repoId uint, filePath string) (bool, error) {

	repoLinks, err := f.RepoLinks.Get(context, repoId)
	if err != nil {
		log.Println(err)
		return false, err
	}

	inDocumentationRepo := strings.HasPrefix(filePath, repoLinks.DocumentationRepo)
	inCodebaseRepo := strings.HasPrefix(filePath, repoLinks.CodebaseRepo)

	return inDocumentationRepo || inCodebaseRepo, nil
}

func (f *FilePermissionsUtil) CheckWritePermissions(context context.Context, repoId uint, filePath string) (bool, error) {

	repoLinks, err := f.RepoLinks.Get(context, repoId)
	if err != nil {
		log.Println(err)
		return false, err
	}

	inDocumentationRepo := strings.HasPrefix(filePath, repoLinks.DocumentationRepo)

	return inDocumentationRepo, nil
}
