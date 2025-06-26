package constants

import (
	"github.com/samber/lo"
)

var pypySites = []string{
	"jd.pypy.moe",
}
var wannaSites = []string{
	"api.udon.dance",
	"api.wannadance.online",
}
var biliSites = []string{
	"www.bilibili.com",
	"b23.tv",
	"api.xin.moe",
}
var allSites = []string{
	// PyPyDance
	"jd.pypy.moe",
	// WannaDance
	"api.udon.dance",
	"api.wannadance.online",
	// BiliBili
	"www.bilibili.com",
	"b23.tv",
	"api.xin.moe",
}

func IsPyPySite(host string) bool {
	return lo.IndexOf(pypySites, host) >= 0
}
func IsWannaSite(host string) bool {
	return lo.IndexOf(wannaSites, host) >= 0
}
func IsBiliSite(host string) bool {
	return lo.IndexOf(biliSites, host) >= 0
}

func AllPyPySites() []string {
	return pypySites
}
func AllWannaSites() []string {
	return wannaSites
}
func AllBiliSites() []string {
	return biliSites
}

func CopyAllSites() []string {
	ret := make([]string, len(allSites))
	copy(ret, allSites)
	return ret
}
