package catalog

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

type CacheService struct {
	service.BaseService[struct{}]

	cacheSvc *cache_manager.Service
}

var cacheIds = []string{"pypy_catalog", "wanna_catalog", "dudu_catalog"}
var cacheSessions = make(map[string]types.CDNFileSession)
var sessionMu sync.RWMutex

func NewService(cacheSvc *cache_manager.Service) *CacheService {
	s := &CacheService{
		cacheSvc: cacheSvc,
	}

	s.SetControl("Catalog Cache", s)

	return s
}

func (s *CacheService) ServiceStart() error {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	for _, id := range cacheIds {
		session, err := s.cacheSvc.CreateSession("catalog")
		if err != nil {
			return err
		}

		err = session.Open(id)
		if err != nil {
			session.Close()
			continue
		}

		cacheSessions[id] = session
	}

	readFromCache()

	return nil
}

func (s *CacheService) ServiceStop() error {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	for id, session := range cacheSessions {
		session.Close()
		delete(cacheSessions, id)
	}

	return nil
}

func (s *CacheService) Enabled() bool {
	return true
}
