package persistence

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cdnMappingLogger = utils.NewLogger("CDN Mapping")

// cdnMappingCache 内存缓存，避免频繁查询数据库
var cdnMappingCache = make(map[string]string) // cdn_filename -> song_id
var cdnMappingMutex sync.RWMutex

const cdnUrlMappingTableSQL = `
CREATE TABLE IF NOT EXISTS cdn_url_mapping (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	song_id TEXT NOT NULL,
	cdn_filename TEXT NOT NULL UNIQUE,
	full_url TEXT,
	updated_at INTEGER
);
`

// InitCdnUrlMapping 初始化CDN映射表
func InitCdnUrlMapping() error {
	_, err := DB.Exec(cdnUrlMappingTableSQL)
	if err != nil {
		return err
	}
	// 加载现有映射到内存缓存
	return loadCdnMappingsToCache()
}

// loadCdnMappingsToCache 从数据库加载映射到内存缓存
func loadCdnMappingsToCache() error {
	cdnMappingMutex.Lock()
	defer cdnMappingMutex.Unlock()

	rows, err := DB.Query("SELECT cdn_filename, song_id FROM cdn_url_mapping")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var filename, songID string
		if err := rows.Scan(&filename, &songID); err != nil {
			continue
		}
		cdnMappingCache[filename] = songID
	}
	return nil
}

// SaveCdnUrlMapping 保存CDN URL到视频ID的映射
func SaveCdnUrlMapping(songID, cdnUrl string) {
	// 从URL中提取文件名
	filename := filepath.Base(cdnUrl)
	if filename == "" || filename == "." {
		return
	}

	cdnMappingMutex.Lock()
	defer cdnMappingMutex.Unlock()

	// 检查是否已存在相同的映射
	if existingID, ok := cdnMappingCache[filename]; ok && existingID == songID {
		return
	}

	// 更新内存缓存
	cdnMappingCache[filename] = songID

	// 异步保存到数据库
	go func() {
		_, err := DB.Exec(
			"INSERT OR REPLACE INTO cdn_url_mapping (song_id, cdn_filename, full_url, updated_at) VALUES (?, ?, ?, ?)",
			songID, filename, cdnUrl, time.Now().Unix(),
		)
		if err != nil {
			cdnMappingLogger.ErrorLn("Failed to save CDN mapping:", err)
		}
	}()
}

// FindSongIDByCdnUrl 根据CDN URL查找对应的视频ID
func FindSongIDByCdnUrl(cdnUrl string) (string, bool) {
	filename := filepath.Base(cdnUrl)
	if filename == "" || filename == "." {
		return "", false
	}
	return FindSongIDByCdnFilename(filename)
}

// FindSongIDByCdnFilename 根据CDN文件名查找对应的视频ID
func FindSongIDByCdnFilename(filename string) (string, bool) {
	cdnMappingMutex.RLock()
	defer cdnMappingMutex.RUnlock()

	songID, ok := cdnMappingCache[filename]
	return songID, ok
}

// IsPyPyCdnUrl 检查是否是PyPy CDN的URL
func IsPyPyCdnUrl(host, path string) bool {
	return strings.Contains(host, "pypy.dance") && strings.HasSuffix(path, ".mp4")
}

// IsWannaCdnUrl 检查是否是WannaDance CDN的URL
func IsWannaCdnUrl(host, path string) bool {
	return (strings.Contains(host, "wannadance") || strings.Contains(host, "udon.dance")) &&
		strings.Contains(path, ".mp4")
}

// IsDuDuCdnUrl 检查是否是DuDu CDN的URL
func IsDuDuCdnUrl(host, path string) bool {
	return strings.Contains(host, "dudufit.dance") && strings.Contains(path, ".mp4")
}
