package utils

import "strings"

func IdentifyRoomBrand(roomName string) string {
	if strings.Contains(roomName, "PyPyDance") {
		return "PyPyDance"
	}
	if strings.Contains(roomName, "WannaDance") {
		return "WannaDance"
	}
	return ""
}
