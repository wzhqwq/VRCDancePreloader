package catalog

import (
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

var pypyCatalogManager = baseManager[raw_song.PyPyDanceSong, PyPyDanceCatalogResponse]{}
var wannaCatalogManager = baseManager[raw_song.WannaDanceSong, WannaDanceCatalogResponse]{}
var duduCatalogManager = baseManager[raw_song.DuDuFitDanceSong, DuDuFitDanceCatalogResponse]{}

func setupManagers(cacheSvc *cache_manager.Service) {
	pypyCatalogManager.setup("pypy_catalog", "PyPyDance Catalog", cacheSvc, processPyPyResponse)
	wannaCatalogManager.setup("wanna_catalog", "WannaDance Catalog", cacheSvc, processWannaResponse)
	duduCatalogManager.setup("dudu_catalog", "DuDuFitDance Catalog", cacheSvc, processDuDuResponse)
}

// PyPyDance

type PyPyDanceCatalogResponse struct {
	Timestamp int                      `json:"timestamp"`
	Groups    []string                 `json:"groups"`
	Songs     []raw_song.PyPyDanceSong `json:"songs"`
}

func processPyPyResponse(data *PyPyDanceCatalogResponse) *Catalog[raw_song.PyPyDanceSong] {
	c := &Catalog[raw_song.PyPyDanceSong]{
		songs:       data.Songs,
		indexMap:    map[int]int{},
		updatedTime: time.Unix(int64(data.Timestamp/1000), 0),
	}

	for i, song := range c.songs {
		if song.Group >= 0 && song.Group < len(data.Groups) {
			song.GroupName = data.Groups[song.Group]
		} else {
			song.GroupName = "Unknown"
		}
		c.indexMap[song.ID] = i
	}

	return c
}

// WannaDance

type WannaDanceCatalogResponse struct {
	Time   string           `json:"time"`
	Groups WannaDanceGroups `json:"groups"`
}
type WannaDanceGroups struct {
	Contents []WannaDanceGroupContent `json:"contents"`
}
type WannaDanceGroupContent struct {
	Name  string                    `json:"groupName"`
	Major string                    `json:"major"`
	Songs []raw_song.WannaDanceSong `json:"songInfos"`
}

func processWannaResponse(data *WannaDanceCatalogResponse) *Catalog[raw_song.WannaDanceSong] {
	updatedTime, _ := time.Parse("20060102150405", data.Time)

	groups := data.Groups.Contents
	for _, group := range groups {
		name := group.Name
		for _, song := range group.Songs {
			song.Group = name
		}
	}

	c := &Catalog[raw_song.WannaDanceSong]{
		songs: slices.Concat(
			lo.Map(groups, func(group WannaDanceGroupContent, _ int) []raw_song.WannaDanceSong {
				return group.Songs
			})...,
		),
		indexMap:    map[int]int{},
		updatedTime: updatedTime,
	}

	for i, song := range c.songs {
		c.indexMap[song.DanceId] = i
	}

	return c
}

// DuDuFitDance

type DuDuFitDanceCatalogResponse struct {
	Timestamp int64                       `json:"timestamp"`
	Groups    []DuDuFitDanceGroup         `json:"groups"`
	Videos    []raw_song.DuDuFitDanceSong `json:"videos"`
}
type DuDuFitDanceGroup struct {
	Name   string `json:"name"`
	Videos []int  `json:"videos"`
}

func processDuDuResponse(data *DuDuFitDanceCatalogResponse) *Catalog[raw_song.DuDuFitDanceSong] {
	c := &Catalog[raw_song.DuDuFitDanceSong]{
		songs:       data.Videos,
		indexMap:    map[int]int{},
		updatedTime: time.Unix(data.Timestamp, 0),
	}

	for i, song := range c.songs {
		song.Group = "Unknown"
		c.indexMap[song.ID] = i
	}

	for _, group := range data.Groups {
		name := group.Name
		for _, id := range group.Videos {
			i, ok := c.indexMap[id]
			if ok {
				c.songs[i].Group = name
			}
		}
	}

	return c
}
