package hongbao

import (
	"fmt"
	"math/rand"
	"time"
)

var lowerCase = []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'}
var upperCase = []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G'}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func pickLetter(letters []byte, length int) string {
	l := len(letters)
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		res[i] = letters[rand.Intn(l)]
	}
	return string(res)
}

func randUserName() string {
	return fmt.Sprintf("%s%d", pickLetter(lowerCase, 3), rand.Int63n(9000000000))
}

func randGroupName() string {
	return fmt.Sprintf("%s%d", pickLetter(upperCase, 3), rand.Int63n(9000000000))
}

func randBirthday() string {
	min := time.Date(1976, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2017, 12, 31, 23, 59, 59, 0, time.UTC).Unix()
	delta := max - min

	sec := min + rand.Int63n(delta)
	return time.Unix(sec, 0).Format("2006-01-02")
}

func randCityID() int {
	return 1 + rand.Intn(339)
}

func randProvinceID() int {
	return 1 + rand.Intn(33)
}
