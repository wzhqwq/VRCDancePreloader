package persistence

import (
	"encoding/json"
	"errors"
	"sync"
)

var localWorlds *LocalWorlds

const worldDataTableSQL = `
CREATE TABLE IF NOT EXISTS world_data (
		world TEXT PRIMARY KEY,
		data TEXT,
    	settings TEXT
);
`

type WorldData struct {
	sync.Mutex

	World    string
	Data     map[string]string
	Settings map[string]string
}

func NewWorldData(world string) *WorldData {
	row := DB.QueryRow("SELECT data, settings FROM world_data WHERE world=?", world)

	dataJson := ""
	settingsJson := ""
	err := row.Scan(&dataJson, &settingsJson)
	if err == nil {
		result := &WorldData{
			World: world,
		}

		err = json.Unmarshal([]byte(dataJson), &result.Data)
		if err == nil {
			result.Data = map[string]string{}
		}

		err = json.Unmarshal([]byte(settingsJson), &result.Settings)
		if err == nil {
			result.Settings = map[string]string{}
		}

		return result
	}

	initialSettings := map[string]string{
		"externalReads": "false",
	}
	initialSettingsJson, err := json.Marshal(initialSettings)
	if err != nil {
		// no way
		panic(err)
	}

	q := "INSERT INTO world_data (world, data, settings) VALUES (?, ?, ?)"
	_, err = DB.Exec(q, world, "{}", initialSettingsJson)
	if err != nil {
		logger.ErrorLn("Failed to add world data:", err)
	}

	return &WorldData{
		World:    world,
		Data:     map[string]string{},
		Settings: initialSettings,
	}
}

func (w *WorldData) Get(key string) (string, error) {
	w.Lock()
	defer w.Unlock()

	if val, ok := w.Data[key]; ok {
		return val, nil
	}
	return "", errors.New("key not found")
}

func (w *WorldData) GetSetting(key string) (string, error) {
	w.Lock()
	defer w.Unlock()

	if val, ok := w.Settings[key]; ok {
		return val, nil
	}
	return "", errors.New("key not found")
}

func (w *WorldData) GetBulk(keys []string) (map[string]string, error) {
	w.Lock()
	defer w.Unlock()

	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	result := map[string]string{}
	for _, key := range keys {
		if val, ok := w.Data[key]; ok {
			result[key] = val
		}
	}

	return result, nil
}

func (w *WorldData) GetAll() map[string]string {
	w.Lock()
	defer w.Unlock()

	return w.Data
}

func (w *WorldData) save() error {
	dataJson, err := json.Marshal(w.Data)
	if err != nil {
		logger.ErrorLn("Failed to save world data:", err)
		return err
	}
	settingsJson, err := json.Marshal(w.Settings)
	if err != nil {
		logger.ErrorLn("Failed to save world data:", err)
		return err
	}
	_, err = DB.Exec("UPDATE world_data SET data=?, settings=? WHERE world=?", dataJson, settingsJson, w.World)
	if err != nil {
		logger.ErrorLn("Failed to save world data:", err)
		return err
	}
	return nil
}

func (w *WorldData) Set(key string, value string) error {
	w.Lock()
	defer w.Unlock()

	w.Data[key] = value
	return w.save()
}

func (w *WorldData) SetSetting(key string, value string) error {
	w.Lock()
	defer w.Unlock()

	w.Settings[key] = value
	return w.save()
}

func (w *WorldData) Del(key string) error {
	w.Lock()
	defer w.Unlock()

	delete(w.Data, key)
	return w.save()
}

func (w *WorldData) Clear() error {
	w.Lock()
	defer w.Unlock()

	w.Data = map[string]string{}
	return w.save()
}

type LocalWorlds struct {
	sync.Mutex

	worlds map[string]*WorldData
}

func (l *LocalWorlds) Get(worldID, key string) (value string, err error) {
	l.Lock()
	defer l.Unlock()

	if world, ok := l.worlds[worldID]; ok {
		return world.Get(key)
	}
	return "", errors.New("world not found")
}

func (l *LocalWorlds) GetBulk(worldID string, keys []string) (map[string]string, error) {
	l.Lock()
	defer l.Unlock()

	if world, ok := l.worlds[worldID]; ok {
		return world.GetBulk(keys)
	}
	return nil, errors.New("world not found")
}

func (l *LocalWorlds) GetAll(worldID string) (map[string]string, error) {
	l.Lock()
	defer l.Unlock()

	if world, ok := l.worlds[worldID]; ok {
		return world.GetAll(), nil
	}
	return nil, errors.New("world not found")
}

func (l *LocalWorlds) CheckAccessibility(worldID string) bool {
	l.Lock()
	defer l.Unlock()

	if world, ok := l.worlds[worldID]; ok {
		v, _ := world.GetSetting("externalReads")
		return v == "true"
	}
	return false
}

func (l *LocalWorlds) CreateOrGetWorld(worldId string) *WorldData {
	l.Lock()
	defer l.Unlock()

	if world, ok := l.worlds[worldId]; ok {
		return world
	}

	world := NewWorldData(worldId)
	l.worlds[worldId] = world
	return world
}

func GetLocalWorlds() *LocalWorlds {
	if localWorlds == nil {
		localWorlds = &LocalWorlds{
			worlds: make(map[string]*WorldData),
		}
	}
	return localWorlds
}
