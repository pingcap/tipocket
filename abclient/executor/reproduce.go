package executor

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/abclient/util"
	"github.com/pingcap/tipocket/abclient/pkg/types"
)

var (
	fileAndLinePattern = regexp.MustCompile(`(.*):([0-9]+)$`)
	successSQLPattern = regexp.MustCompile(`\[SUCCESS\] Exec SQL (.*) success$`)
	failSQLPattern = regexp.MustCompile(`\[FAIL\] Exec SQL (.*) error.*$`)
)

func (e *Executor) reproduce() {
	fileAndLineMatch := fileAndLinePattern.FindStringSubmatch(e.opt.Reproduce)
	if len(fileAndLineMatch) != 3 {
		log.Fatal("invalid input reproduce file must be file:line format")
	}
	if !util.FileExists(fileAndLineMatch[1]) {
		log.Fatalf("file %s not exist or not a file\n", fileAndLineMatch[1])
	}
	line, err := strconv.Atoi(fileAndLineMatch[2])
	if err != nil {
		log.Fatalf("line %s not a number\n", fileAndLineMatch[2])
	}

	e.reproduceFromFile(fileAndLineMatch[1], line)
}

func (e *Executor) reproduceFromFile(file string, endLineNum int) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("error when open file %v\n", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("error when close file %v", err)
		}
	}()

	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		if lineNum == endLineNum {
			break
		}
		line := scanner.Text()
		var m []string
		if strings.HasPrefix(line, "[SUCCESS]") {
			m = successSQLPattern.FindStringSubmatch(line)
		} else if strings.HasPrefix(line, "[FAIL]") {
			m = failSQLPattern.FindStringSubmatch(line)
		} else {
			continue
		}
		if len(m) != 2 {
			continue
		}
		if strings.HasPrefix(strings.Trim(strings.ToLower(m[1]), " "), "select") {
			continue
		}
		e.ch <- &types.SQL{
			SQLType: types.SQLTypeExec,
			SQLStmt: m[1],
		}
		if lineNum % 100 == 0 {
			log.Infof("Progress: %d/%d\n", lineNum, endLineNum)
		}
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeExit,
	}
}
