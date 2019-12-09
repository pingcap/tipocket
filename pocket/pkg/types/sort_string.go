package types

// BySQL implement sort interface for string
type BySQL []string

func (a BySQL) Len() int           { return len(a) }
func (a BySQL) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySQL) Less(i, j int) bool {
	var (
		bi = []byte(a[i])
		bj = []byte(a[j])
	)

	for i := 0; i < min(len(bi), len(bj)); i++ {
		if bi[i] != bj[i] {
			return bi[i] < bj[i]
		}
	}
	return len(bi) < len(bj)
}

func min(a, b int) int {
	if a < b {
			return a
	}
	return b
}
