//go:build !race

package preloader

// raceEnabled reports whether this test binary was built with the race detector.
// See race_enabled_test.go.
const raceEnabled = false
