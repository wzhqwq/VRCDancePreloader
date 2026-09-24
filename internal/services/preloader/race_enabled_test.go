//go:build race

package preloader

// raceEnabled reports whether this test binary was built with the race detector.
//
// The two files that define it exist because a build tag is the only way to ask
// the question: the race detector sets the "race" tag, and nothing at runtime
// exposes it.
const raceEnabled = true
