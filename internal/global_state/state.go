package global_state

var isGui = false

func RunInGui() {
	isGui = true
}

func IsInGui() bool {
	return isGui
}
