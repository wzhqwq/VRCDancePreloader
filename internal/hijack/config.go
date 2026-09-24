package hijack

var limitBandwidth = false

func SetLimitBandwidth(limit bool) {
	limitBandwidth = limit
}

func GetLimitBandwidth() bool {
	return limitBandwidth
}
