package persistence

import (
	"database/sql"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"strings"
	"sync"
)

var currentLocalSongs *LocalSongs

const localSongTableSQL = `
CREATE TABLE IF NOT EXISTS local_song (
    	id TEXT PRIMARY KEY,
    	title TEXT,
    	
    	like INTEGER,
    	skill INTEGER,
                                    
    	is_favorite BOOLEAN,
        in_pypy BOOLEAN
);
`

var localSongTableIndicesSQLs = []string{
	"CREATE INDEX IF NOT EXISTS idx_local_song_is_favorite ON local_song (is_favorite)",
	"CREATE INDEX IF NOT EXISTS idx_local_song_like ON local_song (like)",
	"CREATE INDEX IF NOT EXISTS idx_local_song_skill ON local_song (skill)",
	"CREATE INDEX IF NOT EXISTS idx_local_song_in_pypy ON local_song (in_pypy)",
}

type LocalSongs struct {
	sync.Mutex
	FavoriteMap map[string]struct{}

	em *utils.EventManager[string]
}

func (f *LocalSongs) addEntry(entry *LocalSongEntry) {
	// save to db
	query := "INSERT INTO local_song (id, title, like, skill, is_favorite, in_pypy) VALUES (?, ?, ?, ?, ?, ?)"
	_, err := DB.Exec(query, entry.ID, entry.Title, entry.Like, entry.Skill, entry.IsFavorite, entry.InPypy)
	if err != nil {
		log.Printf("failed to save favorite entry: %v", err)
		return
	}
}

func (f *LocalSongs) getEntry(id string) *LocalSongEntry {
	// load from db
	row := DB.QueryRow("SELECT id, title, like, skill, is_favorite, in_pypy FROM local_song WHERE id = ?", id)

	var entry LocalSongEntry
	err := row.Scan(&entry.ID, &entry.Title, &entry.Like, &entry.Skill, &entry.IsFavorite, &entry.InPypy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		log.Printf("failed to load favorite entry: %v", err)
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
			_, err := DB.Exec("UPDATE local_song SET is_favorite = ? WHERE id = ?", true, id)
			if err != nil {
				log.Printf("failed to update favorite entry: %v", err)
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

			IsFavorite: true,
			InPypy:     true,
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
	_, err := DB.Exec("UPDATE local_song SET is_favorite = ? WHERE id = ?", false, id)
	if err != nil {
		log.Printf("failed to update favorite entry: %v", err)
		return
	}

	delete(f.FavoriteMap, id)
	f.notifySubscribers(id)
}

func (f *LocalSongs) AddLocalSongIfNotExist(id, title string) {
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

			IsFavorite: false,
			InPypy:     false,
		}
		f.addEntry(entry)
	}
}

func (f *LocalSongs) LoadEntries() error {
	f.Lock()
	defer f.Unlock()

	// load from db
	rows, err := DB.Query("SELECT id FROM local_song WHERE is_favorite=true")
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
	var query string
	orderBy := "ORDER BY `" + sortBy + "`"
	if !ascending {
		orderBy += " DESC"
	}
	query = "SELECT id, title, like, skill, is_favorite, in_pypy FROM local_song WHERE is_favorite=true " + orderBy + " LIMIT ? OFFSET ?"
	rows, err := DB.Query(query, pageSize, page*pageSize)
	if err != nil {
		log.Printf("failed to load favorite entries: %v", err)
		return nil
	}
	defer rows.Close()

	var entries []*LocalSongEntry
	for rows.Next() {
		var entry LocalSongEntry
		err = rows.Scan(&entry.ID, &entry.Title, &entry.Like, &entry.Skill, &entry.IsFavorite, &entry.InPypy)
		if err != nil {
			log.Printf("failed to scan favorite entry: %v", err)
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
	err := DB.QueryRow("SELECT COUNT(*) FROM local_song WHERE is_favorite=true").Scan(&total)
	if err != nil {
		log.Printf("failed to load favorite entries: %v", err)
		return 0
	}

	return (total + pageSize - 1) / pageSize
}

func (f *LocalSongs) UpdateFavorite(entry *LocalSongEntry) {
	f.Lock()
	defer f.Unlock()

	// update entry
	_, err := DB.Exec("UPDATE local_song SET like = ?, skill = ? WHERE id = ?", entry.Like, entry.Skill, entry.ID)
	if err != nil {
		log.Printf("failed to update favorite entry: %v", err)
		return
	}
}

func (f *LocalSongs) ToPyPyFavorites() string {
	f.Lock()
	defer f.Unlock()

	// load from db
	query := "SELECT id FROM local_song WHERE is_favorite = ? AND in_pypy = ? AND id LIKE ?"
	rows, err := DB.Query(query, true, true, "pypy_%")
	if err != nil {
		log.Printf("failed to load favorite entries: %v", err)
		return ""
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Printf("failed to scan favorite entry: %v", err)
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

	IsFavorite bool
	InPypy     bool
}

func (e *LocalSongEntry) UpdateLike(like int) {
	e.Like = like
	currentLocalSongs.UpdateFavorite(e)
}
func (e *LocalSongEntry) UpdateSkill(skill int) {
	e.Skill = skill
	currentLocalSongs.UpdateFavorite(e)
}
func (e *LocalSongEntry) UpdateSyncToPypy(b bool) {
	e.InPypy = b
	currentLocalSongs.UpdateFavorite(e)
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
