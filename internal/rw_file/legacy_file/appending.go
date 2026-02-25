package legacy_file

import "errors"

func (f *File) Append(_ []byte) (int, error) {
	return 0, errors.New("not supported")
}
