package trunk

type Fragment struct {
	Start  int64
	Length int64
}

func NewFragment(start int64, length int64) *Fragment {
	return &Fragment{
		Start:  start,
		Length: length,
	}
}

func (f *Fragment) End() int64 {
	return f.Start + f.Length
}

func (f *Fragment) Contains(offset int64) bool {
	return f.Start <= offset && offset < f.End()
}

func (f *Fragment) Includes(offset, length int64) bool {
	return f.Start <= offset && offset+length <= f.End()
}
