package utils

import "os"

func DirectoryExists(path string) bool {
	info, error := os.Stat(path)
	if os.IsNotExist(error) {
		return false
	}
	return true && info.IsDir()
}

func HasGitRepo(path string) bool {
	gitPath := path + string(os.PathSeparator) + ".git"
	info, err := os.Stat(gitPath)
	return err == nil && info.IsDir()
}
