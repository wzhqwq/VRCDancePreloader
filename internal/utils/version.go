package utils

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct {
	Major int
	Minor int
	Patch int

	Alpha bool
	Beta  bool
	Rc    bool

	PrereleaseID int
}

func (v Version) PrereleaseLevel() int {
	if v.Alpha {
		return 0
	}
	if v.Beta {
		return 1
	}
	if v.Rc {
		return 2
	}
	return 3
}

func (v Version) NewerThan(right Version) bool {
	if v.Major > right.Major {
		return true
	}
	if v.Minor > right.Minor {
		return true
	}
	if v.Patch > right.Patch {
		return true
	}
	if v.PrereleaseLevel() > right.PrereleaseLevel() {
		return true
	}
	if v.PrereleaseID > right.PrereleaseID {
		return true
	}

	return false
}

func (v Version) String() string {
	result := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Alpha {
		result = fmt.Sprintf("%s-alpha.%d", result, v.PrereleaseID)
	}
	if v.Beta {
		result = fmt.Sprintf("%s-beta.%d", result, v.PrereleaseID)
	}
	if v.Rc {
		result = fmt.Sprintf("%s-rc.%d", result, v.PrereleaseID)
	}

	return result
}

var versionRegex = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+)|)`)

func ParseVersion(text string) (Version, bool) {
	matches := versionRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return Version{}, false
	}

	major, err := strconv.ParseInt(matches[1], 10, 32)
	if err != nil {
		panic(err)
	}
	minor, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		panic(err)
	}

	ver := Version{
		Major: int(major),
		Minor: int(minor),
	}
	if len(matches) > 3 {
		patch, err := strconv.ParseInt(matches[2], 10, 32)
		if err != nil {
			panic(err)
		}
		ver.Patch = int(patch)
	}

	return ver, true
}

type ShortVersion struct {
	Major int
	Minor int
}

func (v ShortVersion) NewerThan(right ShortVersion) bool {
	if v.Major > right.Major {
		return true
	}
	if v.Minor > right.Minor {
		return true
	}

	return false
}

func (v ShortVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

func ParseShortVersion(text string) (ShortVersion, bool) {
	matches := versionRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ShortVersion{}, false
	}

	major, err := strconv.ParseInt(matches[1], 10, 32)
	if err != nil {
		panic(err)
	}
	minor, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		panic(err)
	}

	ver := ShortVersion{
		Major: int(major),
		Minor: int(minor),
	}

	return ver, true
}
