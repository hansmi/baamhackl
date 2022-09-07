package waryio

type iterSlice []string

var _ StringIter = (*iterSlice)(nil)

func (s *iterSlice) Next() (value string, ok bool) {
	if ok = len(*s) > 0; ok {
		value, *s = (*s)[0], (*s)[1:]
	}

	return
}
