package util

import (
	"fmt"
	"os"
	"time"

	"github.com/juju/errors"

	// "github.com/ngaut/log"
	"github.com/go-sql-driver/mysql"
)

// Version information.
var (
	BuildTS   = "None"
	BuildHash = "None"
)

// PrintInfo prints the octopus version information
func PrintInfo() {
	fmt.Println("Git Commit Hash:", BuildHash)
	fmt.Println("UTC Build Time: ", BuildTS)
}

// ErrorMustSame return error if both error not same
func ErrorMustSame(err1, err2 error) error {
	if err1 == nil && err2 == nil {
		return nil
	}

	if (err1 == nil) != (err2 == nil) {
		return errors.Errorf("error not same, got err1: %v and err2: %v", err1, err2)
	}

	myerr1, ok1 := err1.(*mysql.MySQLError)
	myerr2, ok2 := err2.(*mysql.MySQLError)
	// log.Info("ok status", ok1, ok2)
	if ok1 != ok2 {
		return errors.Errorf("error type not same, if mysql error err1: %t, err2: %t", ok1, ok2)
	}
	// both other type error
	if !ok1 && !ok2 {
		return nil
	}

	if myerr1.Number != myerr2.Number {
		return errors.Errorf("error number not same, got err1: %v and err2 %v", err1, err2)
	}

	return nil
}

// FileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a dir exists
func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CurrentTimeStrAsLog return time format as "[2006/01/02 15:06:02.886 +08:00]"
func CurrentTimeStrAsLog() string {
	return fmt.Sprintf("[%s]", FormatTimeStrAsLog(time.Now()))
}

// FormatTimeStrAsLog format given time as as "[2006/01/02 15:06:02.886 +08:00]"
func FormatTimeStrAsLog(t time.Time) string {
	return t.Format("2006/01/02 15:04:05.000 -07:00")
}
