package types

import (
	"io"

	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
)

type PlayItemI interface {
	UpdateProgress(progress float64)
	UpdateStatus(status constants.Status)
	UpdateIndex(index int)
	Download()
	ToReader() (io.ReadSeekCloser, error)
	GetRendered() *PlayItemRendered
}

type PlayItemRendered struct {
	ID       int
	Title    string
	Group    string
	Adder    string
	Status   string
	Progress float64
	Index    int
}
