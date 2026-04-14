package local_executables

import (
	"context"
	"os"

	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

const denoRepoName = "denoland/deno"
const denoAssetName = "deno-x86_64-pc-windows-msvc.zip"
const denoLocalName = "deno.exe"

func getDenoVersion(ctx context.Context) (utils.Version, bool) {
	v, err := Get("deno").Execute(ctx, "-v")
	if err != nil {
		return utils.Version{}, false
	}
	// deno x.y.z
	return utils.ParseVersion(v)
}

func parseDenoReleaseVersion(release *api.BriefRelease) (utils.Version, bool) {
	// vx.y.z
	v, ok := utils.ParseVersion(release.TagName)
	if ok {
		return v, true
	}

	return utils.ParseVersion(release.ReleaseName)
}

func GetLatestDeno(ctx context.Context) (*api.BriefRelease, error) {
	release, err := api.FindRelease(denoRepoName, denoAssetName, ctx)
	if err != nil {
		return nil, err
	}

	latestVersion, ok := parseDenoReleaseVersion(release)
	if !ok {
		return nil, ErrParsingReleaseVersion
	}

	//release.Compatible = latestVersion.IsCompatibleWith(ytDlpMinimumCompatible, ytDlpMaximumCompatible)
	release.Version = latestVersion.String()

	localVersion, ok := getDenoVersion(ctx)
	if ok && latestVersion.OlderThanOrEqual(localVersion) {
		return nil, nil
	}

	release.LocalVersion = localVersion.String()
	return release, nil
}

func GetLocalDenoInfo(ctx context.Context) BinaryInfo {
	executable, ok := getLocalBinary(denoLocalName)
	if !ok {
		return BinaryInfo{}
	}

	size := int64(0)
	if stat, err := os.Stat(executable); err == nil {
		size = stat.Size()
	}

	v, ok := getDenoVersion(ctx)
	if !ok {
		return BinaryInfo{
			Exists: true,
			Size:   size,
		}
	}

	return BinaryInfo{
		Exists:  true,
		Size:    size,
		Version: v.String(),
	}
}
