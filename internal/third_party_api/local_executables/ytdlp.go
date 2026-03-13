package local_executables

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/api"
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

func getYtDlpVersion(ctx context.Context) (utils.Version, bool) {
	err := raiseIntegrityLevel("vrcdp_yt-dlp.exe")
	if err != nil {
		return utils.Version{}, false
	}
	defer func() {
		err := resumeIntegrityLevel("vrcdp_yt-dlp.exe")
		if err != nil {
			logger.ErrorLn("Failed to resume integrity level of yt-dlp: ", err)
		}
	}()
	v, ok := execVersionCheck("vrcdp_yt-dlp.exe", ctx)
	if !ok {
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

	if len(matches) > 4 {
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

func GetLatestYtDlp(ctx context.Context, channel YtDlpBuildChannel) (*api.BriefRelease, bool) {
	release, err := api.FindRelease("yt-dlp/"+string(channel), "yt-dlp.exe", ctx)
	if err != nil {
		return nil, false
	}

	latestVersion, ok := parseYtDlpReleaseVersion(release)
	if !ok {
		return nil, false
	}

	release.Compatible = latestVersion.IsCompatibleWith(ytDlpMinimumCompatible, ytDlpMaximumCompatible)

	localVersion, ok := getYtDlpVersion(ctx)
	if ok && latestVersion.OlderThanOrEqual(localVersion) {
		return nil, false
	}

	release.LocalVersion = localVersion.DateString()
	return release, true
}

func DownloadYtDlp(release *api.BriefRelease, ctx context.Context, onProgress func(total, downloaded int64)) error {
	return DownloadAndReplace("vrcdp_yt-dlp.exe", release, ctx, onProgress)
}

func printVideoInfoWithYtDlp(url, metaKey string, ctx context.Context) (string, error) {
	executable, ok := getLocalBinary("vrcdp_yt-dlp.exe")
	if !ok {
		return "", errors.New("yt-dlp not found")
	}

	err := raiseIntegrityLevel("vrcdp_yt-dlp.exe")
	if err != nil {
		return "", err
	}
	defer func() {
		err := resumeIntegrityLevel("vrcdp_yt-dlp.exe")
		if err != nil {
			logger.ErrorLn("Failed to resume integrity level of yt-dlp: ", err)
		}
	}()

	tempPath := filepath.Join(custom_fyne.AppDataRoot, "temp")
	err = os.MkdirAll(tempPath, 0755)
	if err != nil {
		return "", err
	}

	var args = []string{
		"-v",
		"--print", metaKey,
		"-f", "mp4[height<=?720]",
		"-P", "temp:" + tempPath,
		"--no-playlist",
		"--no-warnings",
		"--no-check-certificates",
	}
	if proxy := requesting.GetClient(requesting.YouTubeVideo).ProxyUrl; proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, executable, args...)
	output, err := cmd.Output()

	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			logger.InfoLn("stderr:\n" + string(ee.Stderr))
		}

		return "", fmt.Errorf("failed to execute '%s': %v", cmd.String(), err)
	}

	return strings.TrimSpace(string(output)), nil
}

func ResolveVideoUrlWithYtDlp(url string, ctx context.Context) (string, error) {
	return printVideoInfoWithYtDlp(url, "urls", ctx)
}

func GetVideoTitleWithYtDlp(url string, ctx context.Context) (string, error) {
	return printVideoInfoWithYtDlp(url, "title", ctx)
}

func parseDuration(h, m, s string) (time.Duration, error) {
	hours, err := strconv.ParseInt(h, 10, 32)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseInt(m, 10, 32)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
}

func GetDurationWithYtDlp(url string, ctx context.Context) (time.Duration, error) {
	str, err := printVideoInfoWithYtDlp(url, "duration_string", ctx)
	if err != nil {
		return 0, err
	}

	// in [H:][mm:]SS
	segments := strings.Split(str, ":")

	switch len(segments) {
	case 1:
		return parseDuration("0", "0", segments[0])
	case 2:
		return parseDuration("0", segments[0], segments[1])
	case 3:
		return parseDuration(segments[0], segments[1], segments[2])
	default:
		return 0, fmt.Errorf("invalid duration string: %s", str)
	}
}
