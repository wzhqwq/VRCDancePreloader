package service

type Validatable interface {
	Validate() error
}

type ConfigEntry[C Validatable, E any] struct {
	Current func() C
	SaveAll func(C) error

	GetEntry func(C) E
	SetEntry func(*C, E)
}

func (s ConfigEntry[C, E]) Get() E {
	return s.GetEntry(s.Current())
}

func (s ConfigEntry[C, E]) Save(e E) error {
	next := s.Current()

	s.SetEntry(&next, e)

	if err := next.Validate(); err != nil {
		return err
	}

	return s.SaveAll(next)
}
