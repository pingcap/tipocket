package sqlsmith

import "log"

func (s *SQLSmith) debugPrintln(a ...interface{}) {
	if s.debug {
		log.Println(a...)
	}
}

func (s *SQLSmith) debugPrintf(f string, a ...interface{}) {
	if s.debug {
		log.Printf(f, a...)
	}
}
