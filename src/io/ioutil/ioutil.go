// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ioutil implements some I/O utility functions.
// I/O 操作集合函数
package ioutil

import (
	"bytes"
	"io"
	"os"
	"sort"
	"sync"
)

// readAll reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with a specified capacity.
// 从r中读取数据直到error或者EOF发生， 返回以指定容量从内部buffer读取到的数据
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	var buf bytes.Buffer
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}() // 捕获因ErrTooLarge产生的panic， 并以正常错误返回

	if int64(int(capacity)) == capacity { // 主要测试下capacity进行int转换时会不会有丢失精度的问题
		buf.Grow(int(capacity)) // 增加buffer的容量
	}
	_, err = buf.ReadFrom(r) // 从r中读取数据
	return buf.Bytes(), err  // 返回读取的数据
}

// ReadAll reads from r until an error or EOF and returns the data it read.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read from src until EOF, it does not treat an EOF from Read
// as an error to be reported.
// 从r中读取所有的数据。或者返回一个err或EOF。
// 成功调用此函数将会返回err == nil, 并不是 err == EOF。
// 因为ReadAll被定义成从src读取数据直到EOF结束，所以它不会将从读取中获取到的EOF作为一个错误。
func ReadAll(r io.Reader) ([]byte, error) {
	// bytes.Minread
	// MinRead is the minimum slice size passed to a Read call by Buffer.ReadFrom.
	// As long as the Buffer has at least MinRead bytes beyond what is required to hold the contents of r,
	// ReadFrom will not grow the underlying buffer.
	return readAll(r, bytes.MinRead) // MinRead == 512
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
// 通过文件名读取文件内容，成功读取文件返回 err == nil， 并不是 err == EOF
// 因为ReadFile读取整个文件，所以它不会将读取到的EOF当作一个错误看待。
func ReadFile(filename string) ([]byte, error) {
	f, err := os.Open(filename) // 打开文件，并返回一个*File对象，该对象实现了Reader接口
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// HACK It's a good but not certain bet that FileInfo will tell us exactly how much to
	// read, so let's try it but be prepared for the answer to be wrong.
	var n int64 = bytes.MinRead // 因为不知道要申请多大的buffer，所以就用512吧

	if fi, err := f.Stat(); err == nil { // 返回文件信息，如果有错误，一般来说是*PathError ，这里主要是为了获取文件大小
		// As initial capacity for readAll, use Size + a little extra in case Size
		// is zero, and to avoid another allocation after Read has filled the
		// buffer. The readAll call will read into its allocated internal buffer
		// cheaply. If the size was wrong, we'll either waste some space off the end
		// or reallocate as needed, but in the overwhelmingly common case we'll get
		// it just right.
		// 作为readAll的初始容量，在使用Size + 一点额外的内容来防止Size为零的情况，
		// 并避免在读取后再分配已填满的缓冲区。
		// readAll调用将非常轻易地读入其分配的内部缓冲区。如果尺寸不对，
		// 我们要么浪费一些空间,或者根据需要重新分配，但在绝大多数情况下，它刚刚好。
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	return readAll(f, n)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
// 将数据写入到文件中， 如果文件不存在，则以相应的权限创建文件，否则在写入前截断文件。
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite // 未能完全写入数据
	}
	if err1 := f.Close(); err == nil { // 这里为啥不处理err1出错的情况？
		err = err1
	}
	return err
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
// 读取目录，并返回根据文件名排序的列表
func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a ReadCloser with a no-op Close method wrapping
// the provided Reader r.
// 传入一个io.Reader，返回一个io.ReadCloser
// 这个readCloser的Close()操作什么也不做
func NopCloser(r io.Reader) io.ReadCloser {
	return nopCloser{r}
}

type devNull int

// devNull implements ReaderFrom as an optimization so io.Copy to
// ioutil.Discard can avoid doing unnecessary work.
// devNull可以避免一些不必要的工作
var _ io.ReaderFrom = devNull(0)

// do nothing
func (devNull) Write(p []byte) (int, error) {
	return len(p), nil
}

// do nothing
func (devNull) WriteString(s string) (int, error) {
	return len(s), nil
}

var blackHolePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8192) // 8k的黑洞
		return &b
	},
}

func (devNull) ReadFrom(r io.Reader) (n int64, err error) {
	bufp := blackHolePool.Get().(*[]byte)
	readSize := 0
	for {
		readSize, err = r.Read(*bufp)
		n += int64(readSize)
		if err != nil {
			blackHolePool.Put(bufp)
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}

// Discard is an io.Writer on which all Write calls succeed
// without doing anything.
var Discard io.Writer = devNull(0)
