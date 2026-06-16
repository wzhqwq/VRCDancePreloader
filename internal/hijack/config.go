package hijack

var limitBandwidth = false
var maxBandwidth = 25

func SetLimitBandwidth(limit bool) {
	limitBandwidth = limit
}

func SetMaxBandwidth(maxInMbps int) {
	maxBandwidth = maxInMbps
}
