package watcher

import (
	"encoding/json"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"

	"regexp"
)

var pwiRequestRegex = regexp.MustCompile(`^\[VRCX-World] (\{.*})`)

var lastRequests = NewOrderedValues[string]()

var pwiLogger = utils.NewLogger("PWI")

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

		pwiLogger.DebugLnf("Set %s to %s in %s", req.Key, req.Value, world.World)
		break
	case "delete":
		if req.Key == "" {
			return errors.New("key is empty")
		}

		err = world.Del(req.Key)
		if err != nil {
			return err
		}

		pwiLogger.DebugLnf("Delete %s in %s", req.Key, world.World)
		break
	case "delete-all":
		err = world.Clear()
		if err != nil {
			return err
		}

		pwiLogger.DebugLnf("Clear data in %s", world.World)
		break
	case "set-setting":
		if req.Key == "" || req.Value == "" {
			return errors.New("key or value is empty")
		}

		err = world.Set(req.Key, req.Value)
		if err != nil {
			return err
		}

		pwiLogger.DebugLnf("Set %s to %s in the settings of %s", req.Key, req.Value, world.World)
		break
	default:
		return errors.New("invalid request type")
	}

	return nil
}

func checkPWILine(version int32, content []byte) bool {
	if service.IsPWIOn() {
		matches := pwiRequestRegex.FindSubmatch(content)
		if len(matches) > 1 {
			lastRequests.Add(version, string(matches[1]))
			return true
		}
	}
	return false
}

func pwiPostProcess() {
	if service.IsPWIOn() {
		requests := lastRequests.Flush()
		for _, request := range requests {
			err := processPwiLog([]byte(request.value))
			if err != nil {
				logger.ErrorLn("Error while processing PWI request:", err)
			}
		}
	}
}
