package legacy_file

import "errors"

func (f *File) Write(_ []byte) (int, error) {
	return 0, errors.New("not supported")
}
