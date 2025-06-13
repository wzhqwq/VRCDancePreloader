package watcher

import (
	"encoding/json"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
	"log"
)

type pwiRequest struct {
	RequestType   string `json:"requestType"`
	ConnectionKey string `json:"connectionKey"`
	Key           string `json:"key"`
	Value         string `json:"value"`
}

func processPwiLog(data []byte) error {
	var req pwiRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	world, err := service.GetWorldData(req.ConnectionKey)
	if err != nil {
		return err
	}

	switch req.RequestType {
	case "store":
		if req.Key == "" || req.Value == "" {
			return errors.New("key or value is empty")
		}

		err = world.Set(req.Key, req.Value)
		if err != nil {
			return err
		}

		log.Printf("Set %s to %s in %s\n", req.Key, req.Value, world.World)
		break
	case "delete":
		if req.Key == "" {
			return errors.New("key is empty")
		}

		err = world.Del(req.Key)
		if err != nil {
			return err
		}

		log.Printf("Delete %s in %s\n", req.Key, world.World)
		break
	case "delete-all":
		err = world.Clear()
		if err != nil {
			return err
		}

		log.Printf("Clear data in %s\n", world.World)
		break
	case "set-setting":
		if req.Key == "" || req.Value == "" {
			return errors.New("key or value is empty")
		}

		err = world.Set(req.Key, req.Value)
		if err != nil {
			return err
		}

		log.Printf("Set %s to %s in the settings of %s\n", req.Key, req.Value, world.World)
		break
	default:
		return errors.New("invalid request type")
	}

	return nil
}
