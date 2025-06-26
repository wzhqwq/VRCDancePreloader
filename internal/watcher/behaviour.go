package watcher

import (
	"bytes"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"regexp"
)

var enterRoomRegex = regexp.MustCompile(`^Entering Room: (.*)`)
var joinWorldRegex = regexp.MustCompile(`^Joining (wrld_.*):`)

var lastEnteredRoom = NewLastValue("")
var lastWorldId = NewLastValue("")

func checkBehaviourLine(version int32, content []byte, backtrace bool) bool {
	if bytes.HasPrefix(content, []byte("[Behaviour]")) {
		// [Behaviour] Entering Room: PyPyDance
		matches := enterRoomRegex.FindSubmatch(content[12:])
		if len(matches) > 1 {
			roomName := string(matches[1])
			if lastEnteredRoom.Set(version, roomName) && !backtrace {
				if brand := utils.IdentifyRoomBrand(roomName); brand != "" {
					switch brand {
					case "WannaDance":
						forceClearWannaState(version)
						break
					case "PyPyDance":
						forceClearPypyState(version)
						break
					}
				}
			}
			return true
		}

		// [Behaviour] Joining wrld_f20326da-f1ac-45fc-a062-609723b097b1:29406~region(jp)
		matches = joinWorldRegex.FindSubmatch(content[12:])
		if len(matches) > 1 {
			lastWorldId.Set(version, string(matches[1]))
			return true
		}
		return true
	}

	return false
}

func behaviourBacktraceDone() bool {
	return lastEnteredRoom.Get() != "" && lastWorldId.Get() != ""
}

func behaviourPostProcess() {
	enteredRoom := lastEnteredRoom.Get()
	lastEnteredRoom.Reset("")

	if enteredRoom != "" {
		log.Println("Entering room: " + enteredRoom)

		playlist.EnterNewRoom(enteredRoom)
		persistence.SetCurrentRoomName(enteredRoom)
	}

	worldId := lastWorldId.Get()
	lastWorldId.Reset("")

	if worldId != "" {
		service.SetCurrentWorldID(worldId)
	}
}
