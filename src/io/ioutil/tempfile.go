// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
// 随机数状态， 在这里生成随机的临时文件名来保证该文件名不存在。
var rand uint32
var randmu sync.Mutex

// 重新设置随机数种子
func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextRandom() string {
	randmu.Lock()
	r := rand
	if r == 0 { // 如果随机数种子为0， 重新生成
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// TempFile creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting *os.File.
// The filename is generated by taking pattern and adding a random
// string to the end. If pattern includes a "*", the random string
// replaces the last "*".
// If dir is the empty string, TempFile uses the default directory
// for temporary files (see os.TempDir).
// Multiple programs calling TempFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file. It is the caller's responsibility
// to remove the file when no longer needed.
// 在指定的目录创建一个临时文件，用以读写，并返回*os.File对象。
func TempFile(dir, pattern string) (f *os.File, err error) {
	if dir == "" {
		dir = os.TempDir() // 如果没指定dir，则使用系统的临时目录
	}

	var prefix, suffix string
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	nconflict := 0
	for i := 0; i < 10000; i++ { // 如果遇到文件已存在，需要继续循环， 这样循环1W次
		name := filepath.Join(dir, prefix+nextRandom()+suffix)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 { // 冲突超过10次， 需要重新设置种子
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break // 打开文件没有错误，跳出循环
	}
	return
}

// TempDir creates a new temporary directory in the directory dir
// with a name beginning with prefix and returns the path of the
// new directory. If dir is the empty string, TempDir uses the
// default directory for temporary files (see os.TempDir).
// Multiple programs calling TempDir simultaneously
// will not choose the same directory. It is the caller's responsibility
// to remove the directory when no longer needed.
// 在当前的目录生成一个以prefix开头的临时目录
func TempDir(dir, prefix string) (name string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		try := filepath.Join(dir, prefix+nextRandom())
		err = os.Mkdir(try, 0700)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if os.IsNotExist(err) { // 不同于创建临时文件，创建临时目录需要额外的判断
			if _, err := os.Stat(dir); os.IsNotExist(err) { // 创建失败，返回empty，和err
				return "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}
