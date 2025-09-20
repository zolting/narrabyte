package services

import (
	"log"
	"strings"
)

type FilePermissionsUtil struct {
	RepoLinks RepoLinkService
}

func (f *FilePermissionsUtil) CheckReadPermissions(repoId uint, filePath string) (bool, error) {

	repoLinks, err := f.RepoLinks.Get(repoId)
	if err != nil {
		log.Println(err)
		return false, err
	}

	inDocumentationRepo := strings.HasPrefix(filePath, repoLinks.DocumentationRepo)
	inCodebaseRepo := strings.HasPrefix(filePath, repoLinks.CodebaseRepo)

	return inDocumentationRepo || inCodebaseRepo, nil
}

func (f *FilePermissionsUtil) CheckWritePermissions(repoId uint, filePath string) (bool, error) {

	repoLinks, err := f.RepoLinks.Get(repoId)
	if err != nil {
		log.Println(err)
		return false, err
	}

	inDocumentationRepo := strings.HasPrefix(filePath, repoLinks.DocumentationRepo)

	return inDocumentationRepo, nil
}
