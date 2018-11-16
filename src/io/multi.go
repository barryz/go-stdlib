// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io

// 单独定义EOFreader，同样实现了 Reader接口
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, EOF // 返回n个已读取的字节数，和一个EOF错误
}

// 多个readers
type multiReader struct {
	readers []Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 { // 有readers时进入迭代
		// Optimization to flatten nested multiReaders (Issue 13558).
		// 这里如果readers的reader是multiReader本身，需要跳过去
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*multiReader); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF {
			// Use eofReader instead of nil to avoid nil panic
			// after performing flatten (Issue 18232).
			mr.readers[0] = eofReader{} // permit earlier GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF {
			if err == EOF && len(mr.readers) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

// MultiReader returns a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.  If any of the readers
// return a non-nil, non-EOF error, Read will return that error.
// 组合多个readers,并返回一个multiReader示例
func MultiReader(readers ...Reader) Reader {

	r := make([]Reader, len(readers))
	/*
		equals:
			for _, re := range readers {
				r = append(r, re)
			}
	*/
	copy(r, readers)
	return &multiReader{r}
}

type multiWriter struct {
	writers []Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// 使用空标识符判断multiWriter是否实现了stringWriter接口 （编译器检查）
// trick 用法
var _ stringWriter = (*multiWriter)(nil)

func (t *multiWriter) WriteString(s string) (n int, err error) {
	var p []byte // lazily initialized if/when needed
	for _, w := range t.writers {
		if sw, ok := w.(stringWriter); ok { // 如果writer是stringWriter，直接调用writer的WriteString方法写入s
			n, err = sw.WriteString(s)
		} else {
			if p == nil {
				p = []byte(s)
			}
			n, err = w.Write(p) // 如果writer没有实现stringWriter， 则需要将s转换成[]byte，然后调用Write方法
		}
		if err != nil {
			return // 写入时遇到错误，直接返回错误
		}
		if n != len(s) { // 如果写入的字节数不等于s的长度 ，返回ErrShortWrite
			err = ErrShortWrite
			return
		}
	}
	return len(s), nil
}

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Each write is written to each listed writer, one at a time.
// If a listed writer returns an error, that overall write operation
// stops and returns the error; it does not continue down the list.
// 将多个writer组合起来，如果传入的writer是一个multiWriter，则将这个multiWriter中的Writer
// 展开，追加进allWriters列表
func MultiWriter(writers ...Writer) Writer {
	allWriters := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter{allWriters}
}
