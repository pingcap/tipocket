package sqlsmith

import (
	"fmt"
	"time"
)

func (s *SQLSmith) rd (n int) int {
	return s.Rand.Intn(n)
}

func (s *SQLSmith) rdRange (n, m int) int {
	if m < n {
		n, m = m, n
	}
	return n + s.Rand.Intn(m - n)
}

func (s *SQLSmith) rdFloat64() float64 {
	return s.Rand.Float64()
}

func (s *SQLSmith) rdDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2100, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := s.Rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

func (s *SQLSmith) getSubTableName() string {
	name := fmt.Sprintf("ss_sub_%d", s.subTableIndex)
	s.subTableIndex++
	return name
}

func (s *SQLSmith) rdString (length int) string {
	res := ""
	for i := 0; i < length; i++ {
		charCode := s.rdRange(33, 127)
		// char '\' and '"' should be escaped
		if charCode == 92 || charCode == 34 {
			res = fmt.Sprintf("%s%s", res, "\\")
		}
		res = fmt.Sprintf("%s%s", res, string(rune(charCode)))
	}
	return res
}
