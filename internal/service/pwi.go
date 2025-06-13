package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"log"
	"net/http"
	"strings"
	"time"
)

var currentPWIServer *PWIServer
var currentWorldID = ""

type PWIConnection struct {
	worldData     *persistence.WorldData
	connectionKey string
}

func NewPWIConnection(debug bool) (*PWIConnection, error) {
	if debug {
		c := &PWIConnection{
			connectionKey: "12345",
			worldData:     persistence.GetLocalWorlds().CreateOrGetWorld("wrld_12345"),
		}
		return c, nil
	}
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	c := &PWIConnection{
		connectionKey: id.String(),
		worldData:     persistence.GetLocalWorlds().CreateOrGetWorld(currentWorldID),
	}
	return c, nil
}

type PWIServer struct {
	http.Server

	connections map[string]*PWIConnection
	store       *persistence.LocalWorlds

	CurrentConnectionKey string
}

func NewPWIServer() *PWIServer {
	mux := http.NewServeMux()

	s := &PWIServer{
		Server: http.Server{
			Addr:    ":22500",
			Handler: mux,
		},
		connections: make(map[string]*PWIConnection),
		store:       persistence.GetLocalWorlds(),
	}

	s.RegisterHandlers(mux)

	return s
}

func (s *PWIServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/vrcx/status", s.handleStatus)
	mux.HandleFunc("/vrcx/data/init", s.handleInit)
	mux.HandleFunc("/vrcx/data/get", s.handleGet)
	mux.HandleFunc("/vrcx/data/getbulk", s.handleGetBulk)
	mux.HandleFunc("/vrcx/data/getall", s.handleGetAll)
}

func (s *PWIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *PWIServer) updateConnectionKey(debug bool) error {
	if w, ok := s.connections[s.CurrentConnectionKey]; ok {
		if w.worldData.World == currentWorldID {
			return nil
		}
	}

	c, err := NewPWIConnection(debug)
	if err != nil {
		return err
	}

	s.connections[c.connectionKey] = c
	s.CurrentConnectionKey = c.connectionKey

	return nil
}

type commonResp struct {
	OK            bool        `json:"ok"`
	ConnectionKey string      `json:"connectionKey,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Error         string      `json:"error,omitempty"`
}

func (s *PWIServer) handleInit(w http.ResponseWriter, r *http.Request) {
	debug := r.URL.Query().Get("debug") == "true"

	err := s.updateConnectionKey(debug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, commonResp{OK: false, Error: err.Error()})
		return
	}

	resp := commonResp{OK: true, ConnectionKey: s.CurrentConnectionKey, Data: s.CurrentConnectionKey}
	writeJSON(w, http.StatusOK, resp)
}

func (s *PWIServer) handleGet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := q.Get("key")
	world := q.Get("world")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, commonResp{OK: false, Error: "`key` parameter required"})
		return
	}

	err := s.updateConnectionKey(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, commonResp{OK: false, Error: err.Error()})
		return
	}

	var value string
	if world != "" {
		if !s.store.CheckAccessibility(world) {
			writeJSON(w, http.StatusOK, commonResp{OK: false, Error: "`world` does not exist or forbids external reads"})
			return
		}
		value, err = s.store.Get(world, key)
	} else {
		value, err = s.store.Get(currentWorldID, key)
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, commonResp{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, commonResp{OK: true, ConnectionKey: s.CurrentConnectionKey, Data: value})
}

func (s *PWIServer) handleGetBulk(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	keysParam := q.Get("keys")
	world := q.Get("world")
	if keysParam == "" {
		writeJSON(w, http.StatusBadRequest, commonResp{OK: false, Error: "`keys` parameter required"})
		return
	}
	keys := strings.Split(keysParam, ",")

	err := s.updateConnectionKey(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, commonResp{OK: false, Error: err.Error()})
		return
	}

	var m map[string]string
	if world != "" {
		if !s.store.CheckAccessibility(world) {
			writeJSON(w, http.StatusOK, commonResp{OK: false, Error: "`world` does not exist or forbids external reads"})
			return
		}
		m, err = s.store.GetBulk(world, keys)
	} else {
		m, err = s.store.GetBulk(currentWorldID, keys)
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, commonResp{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, commonResp{OK: true, ConnectionKey: s.CurrentConnectionKey, Data: m})
}

func (s *PWIServer) handleGetAll(w http.ResponseWriter, r *http.Request) {
	world := r.URL.Query().Get("world")

	err := s.updateConnectionKey(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, commonResp{OK: false, Error: err.Error()})
		return
	}

	var m map[string]string
	if world != "" {
		if !s.store.CheckAccessibility(world) {
			writeJSON(w, http.StatusOK, commonResp{OK: false, Error: "`world` does not exist or forbids external reads"})
			return
		}
		m, err = s.store.GetAll(world)
	} else {
		m, err = s.store.GetAll(currentWorldID)
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, commonResp{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, commonResp{OK: true, ConnectionKey: s.CurrentConnectionKey, Data: m})
}

func GetWorldData(connectionKey string) (*persistence.WorldData, error) {
	if connectionKey == "" {
		return nil, errors.New("connectionKey is empty")
	}
	if currentPWIServer == nil {
		return nil, errors.New("PWI Service is not running")
	}
	if c, ok := currentPWIServer.connections[connectionKey]; ok {
		return c.worldData, nil
	}
	return nil, errors.New("ConnectionKey is not found")
}

func StartPWIServer() {
	if currentPWIServer != nil {
		return
	}
	log.Println("Starting PWI Server")
	currentPWIServer = NewPWIServer()

	go func() {
		if err := currentPWIServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Println("Error starting PWI Server:", err)
			currentPWIServer = nil
		}
	}()
}

func StopPWIServer() {
	if currentPWIServer != nil {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := currentPWIServer.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
		currentPWIServer = nil
	}
}

func IsPWIOn() bool {
	return currentPWIServer != nil
}

func SetCurrentWorldID(id string) {
	currentWorldID = id
}
