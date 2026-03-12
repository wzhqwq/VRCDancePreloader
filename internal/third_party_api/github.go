package third_party_api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func requestGithubApi[T any](url string, ctx context.Context) (*T, error) {
	client := requesting.GetClient(requesting.GitHubApi)

	req, err := client.NewGetRequest(url, ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("error requesting GitHub API: " + res.Status)
	}

	var resp T
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type GitHubRelease struct {
	ID      int    `json:"id"`
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`

	TargetCommitish string `json:"target_commitish"`

	PublishedAt time.Time `json:"published_at"`

	Assets []GitHubReleaseAsset `json:"assets"`
}
type GitHubReleaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`

	Name        string `json:"name"`
	State       string `json:"state"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Digest      string `json:"digest"`

	CreatedAt time.Time `json:"created_at"`
}

type BriefRelease struct {
	GitHubReleaseAsset

	ID          int
	TagName     string
	ReleaseName string
	ReleaseBody string

	TargetCommitish string

	PublishedAt time.Time

	Compatible   bool
	LocalVersion string
}

func FindRelease(repoName, assetName string, ctx context.Context) (*BriefRelease, error) {
	release, err := requestGithubApi[GitHubRelease](utils.GetLatestReleaseUrl(repoName), ctx)
	if err != nil {
		return nil, err
	}

	asset, found := lo.Find(release.Assets, func(asset GitHubReleaseAsset) bool {
		return asset.Name == assetName
	})
	if !found {
		return nil, errors.New("asset not found")
	}

	brief := &BriefRelease{
		GitHubReleaseAsset: asset,

		ID:          release.ID,
		TagName:     release.TagName,
		ReleaseName: release.Name,
		ReleaseBody: release.Body,

		TargetCommitish: release.TargetCommitish,

		PublishedAt: release.PublishedAt,
	}

	return brief, nil
}
