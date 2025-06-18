package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func GetWannaVideoUrl(id int) string {
	return fmt.Sprintf("http://api.udon.dance/Api/Songs/play?id=%d", id)
}

func CheckWannaUrl(url string) (int, bool) {
	if len(url) < 11 {
		return 0, false
	}

	var id int
	_, err := fmt.Sscanf(url, "http://api.udon.dance/Api/Songs/play?id=%d", &id)
	if err == nil {
		return id, true
	}

	return 0, false
}

func CheckIdIsWanna(id string) (int, bool) {
	if !strings.Contains(id, "wanna_") {
		return 0, false
	}

	num, err := strconv.Atoi(strings.Split(id, "wanna_")[1])
	if err != nil {
		return 0, false
	}

	return num, true
}
