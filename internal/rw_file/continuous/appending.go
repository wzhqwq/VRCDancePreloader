package continuous

func (f *File) Append(bytes []byte) (int, error) {
	err := f.File.AppendTo(f.fragment, bytes)
	if err != nil {
		return 0, err
	}
	f.em.NotifySubscribers(f.fragment.Length)

	if f.File.IsSuffix(f.fragment) {
		f.File.MarkCompleted()
	}

	return len(bytes), nil
}
