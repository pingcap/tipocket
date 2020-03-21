// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"database/sql"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// Used by RandString
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// QueryEntry is a query
type QueryEntry struct {
	Query              string
	Args               []interface{}
	ExpectAffectedRows int64
}

// MustExec must execute sql or fatal
func MustExec(db *sql.DB, query string, args ...interface{}) sql.Result {
	r, err := db.Exec(query, args...)
	if err != nil {
		log.Fatalf("exec %s err %v", query, err)
	}
	return r
}

// ExecWithRollback executes or rollback
func ExecWithRollback(db *sql.DB, queries []QueryEntry) (res sql.Result, err error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, q := range queries {
		res, err = tx.Exec(q.Query, q.Args...)
		if err != nil {
			tx.Rollback()
			return nil, errors.Trace(err)
		}
		if q.ExpectAffectedRows >= 0 {
			affected, err := res.RowsAffected()
			if err != nil {
				tx.Rollback()
				return nil, errors.Trace(err)
			}
			if affected != q.ExpectAffectedRows {
				log.Fatalf("expect affectedRows %v, but got %v, query %v", q.ExpectAffectedRows, affected, q)
			}
		}
	}
	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return nil, errors.Trace(err)
	}
	return
}

func OpenDB(dsn string, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdleConns)
	return db, nil
}

// RunWithRetry tries to run func in specified count
func RunWithRetry(ctx context.Context, retryCnt int, interval time.Duration, f func() error) error {
	var (
		err error
	)
	for i := 0; retryCnt < 0 || i < retryCnt; i++ {
		err = f()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
	return errors.Trace(err)
}

// RandString reference: http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func RandString(b []byte, r *rand.Rand) {
	n := len(b)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, r.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = r.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(alphabet) {
			b[i] = alphabet[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
}

// IsFileExist returns true if the file exists.
func IsFileExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// Wget downloads a string URL to the dest directory and returns the file path.
// SKips if the file already exists.
func Wget(ctx context.Context, rawURL string, dest string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if len(dest) == 0 {
		dest = "."
	}

	fileName := path.Base(u.Path)
	filePath := path.Join(dest, fileName)
	if IsFileExist(filePath) {
		return filePath, nil
	}

	os.MkdirAll(dest, 0755)
	err = exec.CommandContext(ctx, "wget", "--tries", "20", "--waitretry", "60",
		"--retry-connrefused", "--dns-timeout", "60", "--connect-timeout", "60",
		"--read-timeout", "60", "--directory-prefix", dest, rawURL).Run()
	return filePath, errors.Annotatef(err, "wget %s to %s", rawURL, dest)
}

// InstallArchive downloads the URL and extracts the archive to the dest diretory.
// Supports zip, and tarball.
func InstallArchive(ctx context.Context, rawURL string, dest string) error {
	var name string
	var err error
	if strings.HasPrefix(rawURL, "file://") {
		name = strings.Trim(rawURL, "file://")
	} else {
		if name, err = Wget(ctx, rawURL, "/tmp"); err != nil {
			return err
		}
	}

	if strings.HasSuffix(name, ".zip") {
		err = exec.CommandContext(ctx, "unzip", "-d", dest, name).Run()
	} else {
		err = exec.CommandContext(ctx, "tar", "-xf", name, "-C", dest).Run()
	}
	if err != nil {
		return errors.Annotate(err, "decompression error")
	}
	return nil
}
