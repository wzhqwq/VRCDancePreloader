package utils

import "fmt"

func GetLatestReleaseUrl(repo string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
}
