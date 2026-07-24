package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type Manager[T any] interface {
	Refresh()
	OnAvailable()
	Handle() *interactive.RemoteHandle[*Catalog[T]]
}

var retryPolicy = utils.RetryPolicy{
	MaxRetries: 5,
	Delay:      time.Second * 3,
	Jitter:     true,
}

type Catalog[T any] struct {
	songs []T

	indexMap    map[int]int
	updatedTime time.Time
}

func (c *Catalog[T]) FindSong(id int) (T, bool) {
	i, ok := c.indexMap[id]
	if !ok {
		var t T
		return t, false
	}
	return c.songs[i], true
}

func (c *Catalog[T]) UpdatedTime() time.Time {
	return c.updatedTime
}

type baseManager[T any, R any] struct {
	catalogId string

	url    string
	client *requesting.ClientProvider

	statefulCatalog *interactive.RemoteManager[*Catalog[T]]

	availableEm *utils.EventManager[bool]

	processFn func(*R) *Catalog[T]

	logger utils.LoggerImpl

	mu sync.RWMutex
}

func (m *baseManager[T, R]) Refresh() {
	m.statefulCatalog.RefreshActive()
}

func (m *baseManager[T, R]) Handle() *interactive.RemoteHandle[*Catalog[T]] {
	return m.statefulCatalog.Acquire(m.catalogId)
}

func (m *baseManager[T, R]) OnAvailable() {
	m.availableEm.NotifySubscribers(true)
}

func (m *baseManager[T, R]) shutdown() {
	m.statefulCatalog.Close()
}

func (m *baseManager[T, R]) setup(id, name string, processFn func(*R) *Catalog[T]) {
	m.availableEm = utils.NewEventManager[bool]()
	m.logger = utils.NewLogger(name)
	m.processFn = processFn
	m.catalogId = id

	switch id {
	case "pypy_catalog":
		m.url = internal_id.GetPyPyListUrl()
		m.client = requesting.GetClient(requesting.PyPyDance)
	case "wanna_catalog":
		m.url = internal_id.GetWannaListUrl()
		m.client = requesting.GetClient(requesting.WannaDance)
	case "dudu_catalog":
		m.url = internal_id.GetDuDuListUrl()
		m.client = requesting.GetClient(requesting.DuDuFitDance)
	}

	m.statefulCatalog = interactive.NewRemoteManager(m.request, nil, 1, 1)
	scheduler := utils.NewBasicScheduler()
	if id == "pypy_catalog" {
		scheduler = utils.PyPyVideoScheduler()
	}
	m.statefulCatalog.BindScheduler(scheduler)
	m.statefulCatalog.BindAvailability(m.availableEm.SubscribeEvent)
	m.statefulCatalog.BindRetry(&retryPolicy)
	m.statefulCatalog.BindLogger(m.logger)
}

func (m *baseManager[T, R]) readFromCache() {
	sessionMu.RLock()
	defer sessionMu.RUnlock()

	session, ok := cacheSessions[m.catalogId]
	if !ok {
		m.logger.WarnLn("Failed to open catalog cache session, we will fetch and save it in memory")
		return
	}

	file, err := session.AcquireFile()
	if err != nil {
		m.logger.WarnLn("Failed to open catalog cache file, we will fetch and save it in memory")
		return
	}
	defer session.ReleaseFile()

	if !file.IsComplete() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3000000)
	defer cancel()
	var cached R

	err = json.NewDecoder(file.RequestRs(ctx)).Decode(&cached)
	if err != nil {
		m.logger.WarnLn("Failed to decode catalog cache file:", err)
		return
	}

	c := m.processFn(&cached)

	m.statefulCatalog.ModifyPlaceholder(m.catalogId, c)
}

func (m *baseManager[T, R]) saveToCache(bytes []byte) {
	sessionMu.RLock()
	defer sessionMu.RUnlock()

	session, ok := cacheSessions[m.catalogId]
	if !ok {
		return
	}

	file, err := session.AcquireFile()
	if err != nil {
		m.logger.WarnLn("Failed to open catalog cache file")
		return
	}
	defer session.ReleaseFile()

	err = file.Init(int64(len(bytes)), time.Time{})
	if err != nil {
		m.logger.WarnLn("Failed to init catalog cache file")
		return
	}

	_, err = file.Write(bytes)
	if err != nil {
		m.logger.ErrorLn("Failed to copy through cache file:", err)
	} else {
		m.logger.InfoLn("Saved to cache")
	}
}

func (m *baseManager[T, R]) request(_ string, ctx context.Context) (*Catalog[T], error) {
	m.logger.InfoLn("Downloading", m.url)

	//return nil, interactive.ErrUnrecoverable

	req, err := m.client.NewGetRequest(m.url, ctx)
	if err != nil {
		return nil, err
	}

	requesting.SetupHeader(req, m.url)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%wcatalog is temporarily unavailable: %s", interactive.ErrTemporarilyUnavailable, resp.Status)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%wcatalog is not available: %s", interactive.ErrUnrecoverable, resp.Status)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter, _ := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 32)
		return nil, utils.NewThrottledError(time.Duration(retryAfter) * time.Second)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to download, status: " + resp.Status)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, err
	}

	m.saveToCache(buf.Bytes())

	var data R
	dec := json.NewDecoder(&buf)
	err = dec.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("%wfailed to parse catalog response: %w", interactive.ErrUnrecoverable, err)
	}

	return m.processFn(&data), nil
}
