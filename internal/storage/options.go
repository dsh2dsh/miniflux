package storage

type Option func(s *Storage)

func WithNewDedup() Option {
	return func(s *Storage) {
		s.WithDedup(NewDedupEntries())
	}
}

func (s *Storage) WithDedup(d *DedupEntries) *Storage {
	s.dedup = d
	return s
}

func (s *Storage) DedupEntries() *DedupEntries { return s.dedup }
