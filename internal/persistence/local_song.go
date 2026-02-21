package persistence

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vs"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var currentLocalSongs *LocalSongs

var localSongTable = db_vs.DefTable("local_song").DefColumns(
	db_vs.NewTextId(),
	db_vs.NewText("title").SetIndexed(),
	db_vs.NewInt("like").SetIndexed(),
	db_vs.NewInt("skill").SetIndexed(),
	db_vs.NewBool("is_favorite").SetIndexed(),
	db_vs.NewBool("sync_in_game").SetIndexed(),
)

var localSongColumns = []string{"id", "title", "like", "skill", "is_favorite", "sync_in_game"}

var insertLocalSong = localSongTable.Insert(localSongColumns...).Build()

var getLocalSong = localSongTable.Select(localSongColumns...).Where("id = ?").Build()
var listFavoriteIds = localSongTable.Select("id").Where("is_favorite = true").Build()
var getTotalFavorites = localSongTable.Select("COUNT(*)").Where("is_favorite = true").Build()
var listFavorites = localSongTable.Select(localSongColumns...).Where("is_favorite = true").Paginate()

var setFavorite = localSongTable.Update().Set("is_favorite = ?").Where("id = ?").Build()
var setLocalSong = localSongTable.Update().Set("title = ?", "like = ?", "skill = ?").Where("id = ?").Build()

var listFavoriteIdsInPyPy = localSongTable.Select("id").Where("is_favorite = true AND sync_in_game = true AND id LIKE ?").Build()

type LocalSongs struct {
	sync.Mutex
	FavoriteMap map[string]struct{}

	em *utils.EventManager[string]
}

func (f *LocalSongs) addEntry(entry *LocalSongEntry) {
	// save to db
	_, err := localSongTable.Exec(insertLocalSong, entry.ID, entry.Title, entry.Like, entry.Skill, entry.IsFavorite, entry.IsSyncInGame)
	if err != nil {
		logger.ErrorLn("Failed to save favorite entry:", err)
	}
}

func (f *LocalSongs) getEntry(id string) *LocalSongEntry {
	// load from db
	row := localSongTable.QueryRow(getLocalSong, id)

	var entry LocalSongEntry
	err := row.Scan(&entry.ID, &entry.Title, &entry.Like, &entry.Skill, &entry.IsFavorite, &entry.IsSyncInGame)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logger.ErrorLn("Failed to load favorite entry:", err)
		return nil
	}

	return &entry
}

func (f *LocalSongs) SetFavorite(id, title string) {
	f.Lock()
	defer f.Unlock()

	// check if already favorite
	_, ok := f.FavoriteMap[id]
	if ok {
		return
	}

	// check if already exists in db
	entry := f.getEntry(id)
	if entry != nil {
		if !entry.IsFavorite {
			// update is_favorite
			_, err := localSongTable.Exec(setFavorite, true, id)
			if err != nil {
				logger.ErrorLn("Failed to update favorite entry:", err)
				return
			}
		}
	} else {
		// create new entry
		entry = &LocalSongEntry{
			ID:    id,
			Title: title,

			Like:  0,
			Skill: 0,

			IsFavorite:   true,
			IsSyncInGame: false,
		}
		f.addEntry(entry)
	}

	f.FavoriteMap[id] = struct{}{}
	f.notifySubscribers(id)
}

func (f *LocalSongs) UnsetFavorite(id string) {
	f.Lock()
	defer f.Unlock()

	// check if favorite
	_, ok := f.FavoriteMap[id]
	if !ok {
		return
	}

	// update is_favorite
	_, err := localSongTable.Exec(setFavorite, false, id)
	if err != nil {
		logger.ErrorLn("Failed to update favorite entry:", err)
		return
	}

	delete(f.FavoriteMap, id)
	f.notifySubscribers(id)
}

func (f *LocalSongs) AddLocalSongIfNotExist(id, title string) *LocalSongEntry {
	f.Lock()
	defer f.Unlock()

	// check if already exists in db
	entry := f.getEntry(id)
	if entry == nil {
		// create new entry
		entry = &LocalSongEntry{
			ID:    id,
			Title: title,

			Like:  0,
			Skill: 0,

			IsFavorite:   false,
			IsSyncInGame: false,
		}
		f.addEntry(entry)
	}
	return entry
}

func (f *LocalSongs) LoadEntries() error {
	f.Lock()
	defer f.Unlock()

	// load from db
	rows, err := localSongTable.Query(listFavoriteIds)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return err
		}

		f.FavoriteMap[id] = struct{}{}
	}

	return nil
}

func (f *LocalSongs) IsFavorite(id string) bool {
	f.Lock()
	defer f.Unlock()

	_, ok := f.FavoriteMap[id]
	return ok
}

func (f *LocalSongs) ListFavorites(page, pageSize int, sortBy string, ascending bool) []*LocalSongEntry {
	f.Lock()
	defer f.Unlock()

	// load from db
	rows, err := localSongTable.Query(listFavorites.Sort(sortBy, ascending).Build(), pageSize, page*pageSize)
	if err != nil {
		logger.ErrorLn("Failed to load favorite entries:", err)
		return nil
	}
	defer rows.Close()

	var entries []*LocalSongEntry
	for rows.Next() {
		var entry LocalSongEntry
		err = rows.Scan(&entry.ID, &entry.Title, &entry.Like, &entry.Skill, &entry.IsFavorite, &entry.IsSyncInGame)
		if err != nil {
			logger.ErrorLn("Failed to scan favorite entry:", err)
			return nil
		}

		entries = append(entries, &entry)
	}

	return entries
}

func (f *LocalSongs) CalculateTotalPages(pageSize int) int {
	f.Lock()
	defer f.Unlock()

	// load from db
	var total int
	err := localSongTable.QueryRow(getTotalFavorites).Scan(&total)
	if err != nil {
		logger.ErrorLn("Failed to load favorite entries:", err)
		return 0
	}

	return (total + pageSize - 1) / pageSize
}

func (f *LocalSongs) UpdateEntry(entry *LocalSongEntry) {
	f.Lock()
	defer f.Unlock()

	// update entry
	_, err := localSongTable.Exec(setLocalSong, entry.Title, entry.Like, entry.Skill, entry.ID)
	if err != nil {
		logger.ErrorLn("Failed to update entry:", err)
		return
	}
}

func (f *LocalSongs) ToPyPyFavorites() string {
	f.Lock()
	defer f.Unlock()

	// load from db
	rows, err := localSongTable.Query(listFavoriteIdsInPyPy, "pypy_%")
	if err != nil {
		logger.ErrorLn("Failed to load favorite entries:", err)
		return ""
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			logger.ErrorLn("Failed to scan favorite entry:", err)
			return ""
		}

		ids = append(ids, id[5:])
	}

	return strings.Join(ids, ",")
}

type LocalSongEntry struct {
	ID    string
	Title string

	Like  int
	Skill int

	IsFavorite   bool
	IsSyncInGame bool
}

func (e *LocalSongEntry) UpdateLike(like int) {
	e.Like = like
	currentLocalSongs.UpdateEntry(e)
}
func (e *LocalSongEntry) UpdateSkill(skill int) {
	e.Skill = skill
	currentLocalSongs.UpdateEntry(e)
}
func (e *LocalSongEntry) UpdateSyncInGame(b bool) {
	e.IsSyncInGame = b
	currentLocalSongs.UpdateEntry(e)
}
func (e *LocalSongEntry) UpdateTitle(title string) {
	e.Title = title
	currentLocalSongs.UpdateEntry(e)
}
func (e *LocalSongEntry) SetFavorite() {
	currentLocalSongs.SetFavorite(e.ID, e.Title)
}
func (e *LocalSongEntry) UnsetFavorite() {
	currentLocalSongs.UnsetFavorite(e.ID)
}

func InitLocalSongs() {
	currentLocalSongs = &LocalSongs{
		FavoriteMap: make(map[string]struct{}),
		em:          utils.NewEventManager[string](),
	}
	currentLocalSongs.LoadEntries()
}

func GetLocalSongs() *LocalSongs {
	return currentLocalSongs
}

func IsFavorite(id string) bool {
	return currentLocalSongs.IsFavorite(id)
}

func GetEntry(id string) (*LocalSongEntry, error) {
	e := currentLocalSongs.getEntry(id)
	if e == nil {
		return nil, errors.New("entry not found")
	}
	return e, nil
}

func UpdateSavedTitle(id, title string) {
	entry := currentLocalSongs.AddLocalSongIfNotExist(id, title)
	if entry.Title != title {
		entry.UpdateTitle(title)
	}
}
