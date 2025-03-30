package persistence

import (
	"database/sql"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"strings"
	"sync"
)

var currentFavorite *Favorites

const favoriteTableSQL = `
CREATE TABLE IF NOT EXISTS favorite (
    	id TEXT PRIMARY KEY,
    	title TEXT,
    	
    	like INTEGER,
    	skill INTEGER,
                                    
    	is_favorite BOOLEAN,
        in_pypy BOOLEAN
);
`

var favoriteTableIndicesSQLs = []string{
	"CREATE INDEX IF NOT EXISTS idx_favorite_is_favorite ON favorite (is_favorite)",
	"CREATE INDEX IF NOT EXISTS idx_favorite_like ON favorite (like)",
	"CREATE INDEX IF NOT EXISTS idx_favorite_skill ON favorite (skill)",
	"CREATE INDEX IF NOT EXISTS idx_favorite_in_pypy ON favorite (in_pypy)",
}

type Favorites struct {
	sync.Mutex
	Entries map[string]struct{}

	em *utils.StringEventManager
}

func (f *Favorites) addEntry(entry *FavoriteEntry) {
	// save to db
	query := "INSERT INTO favorite (id, title, like, skill, is_favorite, in_pypy) VALUES (?, ?, ?, ?, ?, ?)"
	_, err := DB.Exec(query, entry.ID, entry.Title, entry.Like, entry.Skill, entry.IsFavorite, entry.InPypy)
	if err != nil {
		log.Printf("failed to save favorite entry: %v", err)
		return
	}

	f.Entries[entry.ID] = struct{}{}
}

func (f *Favorites) getEntry(id string) *FavoriteEntry {
	// load from db
	row := DB.QueryRow("SELECT id, title, like, skill, is_favorite, in_pypy FROM favorite WHERE id = ?", id)

	var entry FavoriteEntry
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

func (f *Favorites) SetFavorite(id, title string) {
	f.Lock()
	defer f.Unlock()

	// check if already favorite
	_, ok := f.Entries[id]
	if ok {
		return
	}

	// check if already exists in db
	entry := f.getEntry(id)
	if entry != nil {
		if !entry.IsFavorite {
			// update is_favorite
			_, err := DB.Exec("UPDATE favorite SET is_favorite = ? WHERE id = ?", true, id)
			if err != nil {
				log.Printf("failed to update favorite entry: %v", err)
				return
			}
		}
	} else {
		// create new entry
		entry = &FavoriteEntry{
			ID:    id,
			Title: title,

			Like:  0,
			Skill: 0,

			IsFavorite: true,
			InPypy:     true,
		}
		f.addEntry(entry)
	}

	f.Entries[id] = struct{}{}
	f.notifySubscribers(id)
}

func (f *Favorites) UnsetFavorite(id string) {
	f.Lock()
	defer f.Unlock()

	// check if favorite
	_, ok := f.Entries[id]
	if !ok {
		return
	}

	// update is_favorite
	_, err := DB.Exec("UPDATE favorite SET is_favorite = ? WHERE id = ?", false, id)
	if err != nil {
		log.Printf("failed to update favorite entry: %v", err)
		return
	}

	delete(f.Entries, id)
	f.notifySubscribers(id)
}

func (f *Favorites) LoadEntries() error {
	f.Lock()
	defer f.Unlock()

	// load from db
	rows, err := DB.Query("SELECT id FROM favorite")
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

		f.Entries[id] = struct{}{}
	}

	return nil
}

func (f *Favorites) IsFavorite(id string) bool {
	f.Lock()
	defer f.Unlock()

	_, ok := f.Entries[id]
	return ok
}

func (f *Favorites) ListFavorites(page, pageSize int, sortBy string, ascending bool) []*FavoriteEntry {
	f.Lock()
	defer f.Unlock()

	// load from db
	var query string
	orderBy := "ORDER BY `" + sortBy + "`"
	if !ascending {
		orderBy += " DESC"
	}
	query = "SELECT id, title, like, skill, is_favorite, in_pypy FROM favorite WHERE is_favorite=true " + orderBy + " LIMIT ? OFFSET ?"
	rows, err := DB.Query(query, pageSize, page*pageSize)
	if err != nil {
		log.Printf("failed to load favorite entries: %v", err)
		return nil
	}
	defer rows.Close()

	var entries []*FavoriteEntry
	for rows.Next() {
		var entry FavoriteEntry
		err = rows.Scan(&entry.ID, &entry.Title, &entry.Like, &entry.Skill, &entry.IsFavorite, &entry.InPypy)
		if err != nil {
			log.Printf("failed to scan favorite entry: %v", err)
			return nil
		}

		entries = append(entries, &entry)
	}

	return entries
}

func (f *Favorites) CalculateTotalPages(pageSize int) int {
	f.Lock()
	defer f.Unlock()

	// load from db
	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM favorite WHERE is_favorite=true").Scan(&total)
	if err != nil {
		log.Printf("failed to load favorite entries: %v", err)
		return 0
	}

	return (total + pageSize - 1) / pageSize
}

func (f *Favorites) UpdateFavorite(entry *FavoriteEntry) {
	f.Lock()
	defer f.Unlock()

	// update entry
	_, err := DB.Exec("UPDATE favorite SET like = ?, skill = ? WHERE id = ?", entry.Like, entry.Skill, entry.ID)
	if err != nil {
		log.Printf("failed to update favorite entry: %v", err)
		return
	}
}

func (f *Favorites) ToPyPyFavorites() string {
	f.Lock()
	defer f.Unlock()

	// load from db
	query := "SELECT id FROM favorite WHERE is_favorite = ? AND in_pypy = ? AND id LIKE ?"
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

type FavoriteEntry struct {
	ID    string
	Title string

	Like  int
	Skill int

	IsFavorite bool
	InPypy     bool
}

func (e *FavoriteEntry) UpdateLike(like int) {
	e.Like = like
	currentFavorite.UpdateFavorite(e)
}
func (e *FavoriteEntry) UpdateSkill(skill int) {
	e.Skill = skill
	currentFavorite.UpdateFavorite(e)
}
func (e *FavoriteEntry) UpdateSyncToPypy(b bool) {
	e.InPypy = b
	currentFavorite.UpdateFavorite(e)
}
func (e *FavoriteEntry) SetFavorite() {
	currentFavorite.SetFavorite(e.ID, e.Title)
}
func (e *FavoriteEntry) UnsetFavorite() {
	currentFavorite.UnsetFavorite(e.ID)
}

func InitFavorites() {
	currentFavorite = &Favorites{
		Entries: make(map[string]struct{}),
		em:      utils.NewStringEventManager(),
	}
	currentFavorite.LoadEntries()
}

func GetFavorite() *Favorites {
	return currentFavorite
}

func IsFavorite(id string) bool {
	return currentFavorite.IsFavorite(id)
}
