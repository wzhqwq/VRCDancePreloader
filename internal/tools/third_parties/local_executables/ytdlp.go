package local_executables

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var ytDlpMaximumCompatible = utils.Version{Major: 2026, Minor: 3, Patch: 3}
var ytDlpMinimumCompatible = utils.Version{Major: 2026, Minor: 1, Patch: 31}

var ytDlpReleaseRegex = regexp.MustCompile(`yt-dlp (?:\S+\s|)([0-9]{4}\.[0-9]{2}\.[0-9]{2}(?:\.\d+|))`)
var ytDlpVersionRegex = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)(?:\.(\d+)|)`)

type YtDlpBuildChannel string

const (
	YtDlpStable  YtDlpBuildChannel = "yt-dlp"
	YtDlpNightly YtDlpBuildChannel = "yt-dlp-nightly-builds"
	YtDlpMaster  YtDlpBuildChannel = "yt-dlp-master-builds"
)

const ytDlpAssetName = "yt-dlp.exe"
const ytDlpLocalName = "y-t-d-l-p.exe"

func getYtDlpVersion(ctx context.Context) (utils.Version, bool) {
	v, err := Get("ytdlp").Execute(ctx, "--version")
	if err != nil {
		return utils.Version{}, false
	}
	// Although the version of yt-dlp is in date format, semantic version is still compatible with it
	return utils.ParseVersion(v)
}

func parseYtDlpVersion(version string) (utils.Version, bool) {
	matches := ytDlpVersionRegex.FindStringSubmatch(version)

	if len(matches) < 4 {
		return utils.Version{}, false
	}

	year, err := strconv.ParseInt(matches[1], 10, 32)
	if err != nil {
		panic(err)
	}
	month, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		panic(err)
	}
	day, err := strconv.ParseInt(matches[3], 10, 32)
	if err != nil {
		panic(err)
	}

	ver := utils.Version{
		Major: int(year),
		Minor: int(month),
		Patch: int(day),
	}

	if len(matches) > 4 && len(matches[4]) > 0 {
		build, err := strconv.ParseInt(matches[4], 10, 32)
		if err != nil {
			panic(err)
		}
		ver.PrereleaseID = int(build)
		ver.Alpha = true
	}

	return ver, true
}

func parseYtDlpReleaseVersion(release *api.BriefRelease) (utils.Version, bool) {
	v, ok := parseYtDlpVersion(release.TagName)
	if ok {
		return v, true
	}

	matches := ytDlpReleaseRegex.FindStringSubmatch(release.ReleaseName)
	if len(matches) != 2 {
		return utils.Version{}, false
	}

	return parseYtDlpVersion(matches[1])
}

func GetLatestYtDlp(ctx context.Context, channel YtDlpBuildChannel) (*api.BriefRelease, error) {
	release, err := api.FindRelease("yt-dlp/"+string(channel), ytDlpAssetName, ctx)
	if err != nil {
		return nil, err
	}

	latestVersion, ok := parseYtDlpReleaseVersion(release)
	if !ok {
		return nil, ErrParsingReleaseVersion
	}

	release.Compatible = latestVersion.IsCompatibleWith(ytDlpMinimumCompatible, ytDlpMaximumCompatible)
	release.Version = latestVersion.DateString()

	localVersion, ok := getYtDlpVersion(ctx)
	if ok && latestVersion.OlderThanOrEqual(localVersion) {
		return nil, nil
	}

	release.LocalVersion = localVersion.DateString()
	return release, nil
}

func GetLocalYtDlpInfo(ctx context.Context) BinaryInfo {
	executable, ok := getLocalBinary(ytDlpLocalName)
	if !ok {
		return BinaryInfo{}
	}

	size := int64(0)
	if stat, err := os.Stat(executable); err == nil {
		size = stat.Size()
	}

	v, ok := getYtDlpVersion(ctx)
	if !ok {
		return BinaryInfo{
			Exists: true,
			Size:   size,
		}
	}

	return BinaryInfo{
		Exists:  true,
		Version: v.DateString(),
		Size:    size,
	}
}

func printVideoInfoWithYtDlp(url, metaKey string, ctx context.Context) (string, error) {
	tempPath := filepath.Join(custom_fyne.AppDataRoot, "temp")
	err := os.MkdirAll(tempPath, 0755)
	if err != nil {
		return "", err
	}

	ytdlpExecutable := Get("ytdlp")
	denoExecutable := Get("deno")

	var args = []string{
		"-v",
		"--print", metaKey,
		"-f", "mp4[height<=?720][protocol^=http]",
		"-P", "temp:" + tempPath,
		"--no-playlist",
		"--no-warnings",
		"--no-check-certificates",
	}
	if proxy := requesting.GetClient(requesting.YouTubeVideo).ProxyUrl; proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	//err = denoExecutable.RequestRunnableIntegrity(ctx)
	//defer denoExecutable.ReleaseRunnableIntegrity()
	err = denoExecutable.RequestRunnable()
	defer denoExecutable.ReleaseRunnable()
	if err != nil && !errors.Is(err, ErrExecutableNotFound) {
		return "", err
	}
	if err == nil {
		args = append(args, "--js-runtimes", "deno:"+denoExecutable.Path)
	}

	args = append(args, url)

	output, err := ytdlpExecutable.Execute(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func ResolveVideoUrlWithYtDlp(url string, ctx context.Context) (string, error) {
	return printVideoInfoWithYtDlp(url, "urls", ctx)
}

func GetVideoThumbnailWithYtDlp(url string, ctx context.Context) (string, error) {
	return printVideoInfoWithYtDlp(url, "thumbnail", ctx)
}

func GetVideoBasicInfoWithYtDlp(url string, ctx context.Context) (*types.GeneralVideoInfo, error) {
	linesString, err := printVideoInfoWithYtDlp(url, "title,duration,channel", ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(linesString, "\n")
	if len(lines) < 3 {
		return nil, errors.New("invalid yt-dlp output")
	}

	duration, err := strconv.ParseInt(lines[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &types.GeneralVideoInfo{
		Title:     lines[0],
		Duration:  time.Duration(duration),
		GroupName: lines[2],
	}, nil
}
