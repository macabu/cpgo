package gh

import "strings"

type Repository struct {
	Org  string
	Name string
}

// ParseRepoURL from the format https://github.com/your-org/your-repo.
func ParseRepoURL(repoURL string) Repository {
	_, orgWithName, _ := strings.Cut(repoURL, "github.com/")

	org, name, _ := strings.Cut(orgWithName, "/")

	return Repository{
		Org:  org,
		Name: strings.TrimSuffix(name, "/"),
	}
}
