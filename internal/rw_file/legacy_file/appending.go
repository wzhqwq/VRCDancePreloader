package legacy_file

import (
	"log"
)

func (f *File) Append(bytes []byte) (int, error) {
	f.fileMutex.RLock()
	defer func() {
		f.fileMutex.RUnlock()

		if f.downloaded == f.totalLen {
			go func() {
				err := f.Save()
				if err != nil {
					log.Printf("Failed to rename file: %s, err: %s", f.baseName, err)
				}
			}()
		}
	}()

	n, err := f.file.WriteAt(bytes, f.downloaded)
	if err != nil {
		return 0, err
	}
	f.downloaded += int64(n)
	f.em.NotifySubscribers(f.downloaded)

	return n, nil
}
