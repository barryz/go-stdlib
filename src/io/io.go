// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package io provides basic interfaces to I/O primitives.
// Its primary job is to wrap existing implementations of such primitives,
// such as those in package os, into shared public interfaces that
// abstract the functionality, plus some other related primitives.
//
// Because these interfaces and primitives wrap lower-level operations with
// various implementations, unless otherwise informed clients should not
// assume they are safe for parallel execution.
package io

import (
	"errors"
)

// Seek whence values.
const (
	SeekStart   = 0 // 从文件开头开始seek
	SeekCurrent = 1 // 从当前位置开始seek
	SeekEnd     = 2 // 从文件结尾处开始seek
)

// ErrShortWrite means that a write accepted fewer bytes than requested
// but failed to return an explicit error.
// 表示写入时接收到的字节少于请求的字节数，但未返回一个明确的错误
var ErrShortWrite = errors.New("short write")

// ErrShortBuffer means that a read required a longer buffer than was provided.
var ErrShortBuffer = errors.New("short buffer")

// EOF is the error returned by Read when no more input is available.
// Functions should return EOF only to signal a graceful end of input.
// If the EOF occurs unexpectedly in a structured data stream,
// the appropriate error is either ErrUnexpectedEOF or some other error
// giving more detail.
// End of File 错误，表示读取完毕
var EOF = errors.New("EOF")

// ErrUnexpectedEOF means that EOF was encountered in the
// middle of reading a fixed-size block or data structure.
// 在固定大小的块或数据结构中读取到一半时遇到一个EOF错误
var ErrUnexpectedEOF = errors.New("unexpected EOF")

// ErrNoProgress is returned by some clients of an io.Reader when
// many calls to Read have failed to return any data or error,
// usually the sign of a broken io.Reader implementation.
var ErrNoProgress = errors.New("multiple Read calls return no data or error")

// Reader is the interface that wraps the basic Read method.
//
// Read reads up to len(p) bytes into p. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered. Even if Read
// returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally
// returns what is available instead of waiting for more.
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of
// bytes read. It may return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil. The next Read should
// return 0, EOF.
//
// Callers should always process the n > 0 bytes returned before
// considering the error err. Doing so correctly handles I/O errors
// that happen after reading some bytes and also both of the
// allowed EOF behaviors.
//
// Implementations of Read are discouraged from returning a
// zero byte count with a nil error, except when len(p) == 0.
// Callers should treat a return of 0 and nil as indicating that
// nothing happened; in particular it does not indicate EOF.
//
// Implementations must not retain p.
// 包含了Read操作的最基本的Reader接口
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Writer is the interface that wraps the basic Write method.
//
// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
//
// Implementations must not retain p.
// 包含了Write操作的最基本的Writer接口
type Writer interface {
	Write(p []byte) (n int, err error)
}

// Closer is the interface that wraps the basic Close method.
//
// The behavior of Close after the first call is undefined.
// Specific implementations may document their own behavior.
type Closer interface {
	Close() error
}

// Seeker is the interface that wraps the basic Seek method.
//
// Seek sets the offset for the next Read or Write to offset,
// interpreted according to whence:
// SeekStart means relative to the start of the file,
// SeekCurrent means relative to the current offset, and
// SeekEnd means relative to the end.
// Seek returns the new offset relative to the start of the
// file and an error, if any.
//
// Seeking to an offset before the start of the file is an error.
// Seeking to any positive offset is legal, but the behavior of subsequent
// I/O operations on the underlying object is implementation-dependent.
// Seek方法设定了下一次读写的位置
// 这里的whence最好使用在文件开头定义的SEEK开头的几个常量
type Seeker interface {
	Seek(offset int64, whence int) (int64, error)
}

// ReadWriter is the interface that groups the basic Read and Write methods.
// ReaderWriter接口组合，一个接口内嵌了另一个接口，即“继承”了该接口的方法集
type ReadWriter interface {
	Reader
	Writer
}

// ReadCloser is the interface that groups the basic Read and Close methods.
// Reader和Closer接口的组合
type ReadCloser interface {
	Reader
	Closer
}

// WriteCloser is the interface that groups the basic Write and Close methods.
// Writer和Closer接口组合
type WriteCloser interface {
	Writer
	Closer
}

// ReadWriteCloser is the interface that groups the basic Read, Write and Close methods.
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ReadSeeker is the interface that groups the basic Read and Seek methods.
type ReadSeeker interface {
	Reader
	Seeker
}

// WriteSeeker is the interface that groups the basic Write and Seek methods.
type WriteSeeker interface {
	Writer
	Seeker
}

// ReadWriteSeeker is the interface that groups the basic Read, Write and Seek methods.
type ReadWriteSeeker interface {
	Reader
	Writer
	Seeker
}

// ReaderFrom is the interface that wraps the ReadFrom method.
//
// ReadFrom reads data from r until EOF or error.
// The return value n is the number of bytes read.
// Any error except io.EOF encountered during the read is also returned.
//
// The Copy function uses ReaderFrom if available.
// ReaderFrom 从r中读取数据，直到遇到EOF或 一个错误为止 ， 返回读取的字节数n
// Copy函数会在必要的时候使用RReaderFrom
type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

// WriterTo is the interface that wraps the WriteTo method.
//
// WriteTo writes data to w until there's no more data to write or
// when an error occurs. The return value n is the number of bytes
// written. Any error encountered during the write is also returned.
//
// The Copy function uses WriterTo if available.
// WriterTo 将数据写入w， 直到遇到EOF或一个错误为止， 返回写入的字节数n
// Copy函数会在必要的时候使用WriterTo
type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

// ReaderAt is the interface that wraps the basic ReadAt method.
//
// ReadAt reads len(p) bytes into p starting at offset off in the
// underlying input source. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered.
//
// When ReadAt returns n < len(p), it returns a non-nil error
// explaining why more bytes were not returned. In this respect,
// ReadAt is stricter than Read.
//
// Even if ReadAt returns n < len(p), it may use all of p as scratch
// space during the call. If some data is available but not len(p) bytes,
// ReadAt blocks until either all the data is available or an error occurs.
// In this respect ReadAt is different from Read.
//
// If the n = len(p) bytes returned by ReadAt are at the end of the
// input source, ReadAt may return either err == EOF or err == nil.
//
// If ReadAt is reading from an input source with a seek offset,
// ReadAt should not affect nor be affected by the underlying
// seek offset.
//
// Clients of ReadAt can execute parallel ReadAt calls on the
// same input source.
//
// Implementations must not retain p.
// 从off(set)处开始读取bytes到p中
type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
}

// WriterAt is the interface that wraps the basic WriteAt method.
//
// WriteAt writes len(p) bytes from p to the underlying data stream
// at offset off. It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// WriteAt must return a non-nil error if it returns n < len(p).
//
// If WriteAt is writing to a destination with a seek offset,
// WriteAt should not affect nor be affected by the underlying
// seek offset.
//
// Clients of WriteAt can execute parallel WriteAt calls on the same
// destination if the ranges do not overlap.
//
// Implementations must not retain p.
type WriterAt interface {
	WriteAt(p []byte, off int64) (n int, err error)
}

// ByteReader is the interface that wraps the ReadByte method.
//
// ReadByte reads and returns the next byte from the input or
// any error encountered. If ReadByte returns an error, no input
// byte was consumed, and the returned byte value is undefined.
// ReadByte读取并返回输入中的下一个字节，如果有error，则返回error。
// 如果返回了error， 返回的byte是未定义的， 不能使用
type ByteReader interface {
	ReadByte() (byte, error)
}

// ByteScanner is the interface that adds the UnreadByte method to the
// basic ReadByte method.
//
// UnreadByte causes the next call to ReadByte to return the same byte
// as the previous call to ReadByte.
// It may be an error to call UnreadByte twice without an intervening
// call to ReadByte.
// ByteScanner 继承了ByteReader，并添加了UnreadByte() error 方法，
// 如果在ReadByte调用之后在调用UnreadByte，则会返回上次ReadByte读取的相同的字节。
// 连续调用两次UnreadByte()方法或者未在ReadByte()之后调用UnreadByte()都会发生错误。
type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

// ByteWriter is the interface that wraps the WriteByte method.
// ByteWriter写入一个字节
type ByteWriter interface {
	WriteByte(c byte) error
}

// RuneReader is the interface that wraps the ReadRune method.
//
// ReadRune reads a single UTF-8 encoded Unicode character
// and returns the rune and its size in bytes. If no character is
// available, err will be set.
// ReadRune()读取一个UTF-8编码的uniconde字符，并返回读取到的符文和该符文的字节大小。
// 如果没读取到，则返回一个错误。
type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

// RuneScanner is the interface that adds the UnreadRune method to the
// basic ReadRune method.
//
// UnreadRune causes the next call to ReadRune to return the same rune
// as the previous call to ReadRune.
// It may be an error to call UnreadRune twice without an intervening
// call to ReadRune.
// RuneScanner 用法和 ByteScanner类似
type RuneScanner interface {
	RuneReader
	UnreadRune() error
}

// stringWriter is the interface that wraps the WriteString method.
type stringWriter interface {
	WriteString(s string) (n int, err error)
}

// WriteString writes the contents of the string s to w, which accepts a slice of bytes.
// If w implements a WriteString method, it is invoked directly.
// Otherwise, w.Write is called exactly once.
// 将字符串s的内容写入到w中，同时也可以接收一个字节切片.
// 如果w实现了WriteString方法，那么就直接调用这个方法，否则
// 会调用w的Write方法，如果直接调用Write，会有一次字符串s到
// 字节切片的转换。
func WriteString(w Writer, s string) (n int, err error) {
	if sw, ok := w.(stringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

// ReadAtLeast reads from r into buf until it has read at least min bytes.
// It returns the number of bytes copied and an error if fewer bytes were read.
// The error is EOF only if no bytes were read.
// If an EOF happens after reading fewer than min bytes,
// ReadAtLeast returns ErrUnexpectedEOF.
// If min is greater than the length of buf, ReadAtLeast returns ErrShortBuffer.
// On return, n >= min if and only if err == nil.
// If r returns an error having read at least min bytes, the error is dropped.
// 从r中至少读取min个字节到buf中
func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min { // buf太小，不能完全写入，返回错误
		return 0, ErrShortBuffer
	}
	for n < min && err == nil { // 逐个字节读取， 当 n == min 或 err != nil 时停止
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min { // 读取正常， err == nil
		err = nil
	} else if n > 0 && err == EOF { // 读取过程中出现EOF错误，返回ErrUnexpectedEOF
		err = ErrUnexpectedEOF
	}
	return
}

// ReadFull reads exactly len(buf) bytes from r into buf.
// It returns the number of bytes copied and an error if fewer bytes were read.
// The error is EOF only if no bytes were read.
// If an EOF happens after reading some but not all the bytes,
// ReadFull returns ErrUnexpectedEOF.
// On return, n == len(buf) if and only if err == nil.
// If r returns an error having read at least len(buf) bytes, the error is dropped.
// 将len(buf)个字节全部读入buf中，内部调用了ReadAtLeast方法
func ReadFull(r Reader, buf []byte) (n int, err error) {
	return ReadAtLeast(r, buf, len(buf))
}

// CopyN copies n bytes (or until an error) from src to dst.
// It returns the number of bytes copied and the earliest
// error encountered while copying.
// On return, written == n if and only if err == nil.
//
// If dst implements the ReaderFrom interface,
// the copy is implemented using it.
// 从src中拷贝n个字节到dst中
func CopyN(dst Writer, src Reader, n int64) (written int64, err error) {
	written, err = Copy(dst, LimitReader(src, n))
	if written == n {
		return n, nil // 拷贝正常，返回
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		// src过早返回，我们必须返回EOF
		err = EOF
	}
	return
}

// Copy copies from src to dst until either EOF is reached
// on src or an error occurs. It returns the number of bytes
// copied and the first error encountered while copying, if any.
//
// A successful Copy returns err == nil, not err == EOF.
// Because Copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
//
// If src implements the WriterTo interface,
// the copy is implemented by calling src.WriteTo(dst).
// Otherwise, if dst implements the ReaderFrom interface,
// the copy is implemented by calling dst.ReadFrom(src).
// 从src拷贝数据到dst，当src达到EOF或者出现错误时停止。
// err == nil 表示拷贝正常， 如果 err == EOF 表示有错误。
// 如果src实现了WriterTo接口，则内部将会将调用src.WriteTo(dst)
// 否则，如果dst实现了ReaderFrom接口，内部将会调用dst.ReadFrom(src)
func Copy(dst Writer, src Reader) (written int64, err error) {
	return copyBuffer(dst, src, nil)
}

// CopyBuffer is identical to Copy except that it stages through the
// provided buffer (if one is required) rather than allocating a
// temporary one. If buf is nil, one is allocated; otherwise if it has
// zero length, CopyBuffer panics.
// 指定一个buf来执行Copy， 默认的Copy是32kb大小
// 比如我们可以指定64kb大小的buf来执行拷贝  CopyBuffer(dst, src, make([]byte, 64 * 1024))
func CopyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 { // buf为空，为空不是nil !!!
		panic("empty buffer in io.CopyBuffer")
	}
	return copyBuffer(dst, src, buf)
}

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
// Copy和CopyBuffer实际内部实现
func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	// 如果reader有WriteTo方法，直接调用该方法可避免一次分配和拷贝
	if wt, ok := src.(WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	// 同样的， 如果writer有ReadFrom方法，直接调用
	if rt, ok := dst.(ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	if buf == nil {
		size := 32 * 1024 // 32kb
		// 如果src实现了LimitedReader接口，size 大小应该等于l.N的大小， 如果l.N大于
		// 32k,我们就使用32k大小的[]byte
		if l, ok := src.(*LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = ErrShortWrite // 表示写入时的字节少于读取的字节数，且没有其他明确错误
				break
			}
		}
		if er != nil { // 从src读取出错 ，不管是EOF还是其他都break
			if er != EOF { // 错误不是EOF时，记录下err
				err = er
			}
			break
		}
	}
	return written, err
}

// LimitReader returns a Reader that reads from r
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedReader.
// 返回一个从r中读取n个字节的新的LimitedReader实例，该实例对应的类型实现了
// Reader接口
func LimitReader(r Reader, n int64) Reader { return &LimitedReader{r, n} }

// A LimitedReader reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns EOF when N <= 0 or when the underlying R returns EOF.
// 从R中读取数据，但是在N的时候就返回。
// 每调用一次Read都将更新N来反应新的剩余的数量。
type LimitedReader struct {
	R Reader // underlying reader
	N int64  // max bytes remaining
}

// 实现Reader接口
func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 { // limit小于等于0， 返回EOF，表示读取完毕
		return 0, EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N] // 直接截取l.N个字节
	}
	n, err = l.R.Read(p) // 将数据读取进p中
	l.N -= int64(n)      // 这里有len(p) =< l.N的情况， 所以每次调用Read后要更新l.N； 如果是len(p) > l.N ， l.N就一次性减为0
	return
}

// NewSectionReader returns a SectionReader that reads from r
// starting at offset off and stops with EOF after n bytes.
// 从r中以offset开始读取字节，直到读取n个为止，并返回EOF
func NewSectionReader(r ReaderAt, off int64, n int64) *SectionReader {
	return &SectionReader{r, off, off, off + n}
}

// SectionReader implements Read, Seek, and ReadAt on a section
// of an underlying ReaderAt.
type SectionReader struct {
	r     ReaderAt
	base  int64
	off   int64
	limit int64
}

// 实现Reader接口
func (s *SectionReader) Read(p []byte) (n int, err error) {
	if s.off >= s.limit { // 偏移量超过最大限制，函数正常结束
		return 0, EOF
	}
	if max := s.limit - s.off; int64(len(p)) > max {
		p = p[0:max] // len(p)大于要读取的max字节数， 切片截取
	}
	n, err = s.r.ReadAt(p, s.off) // 从reader的offset处开始读取到p中
	s.off += int64(n)             // 每次读取完毕，需要更新offset
	return
}

var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

// 实现Seeker接口
func (s *SectionReader) Seek(offset int64, whence int) (int64, error) {
	// s.base的初始值和s.offset相同
	switch whence {
	default:
		return 0, errWhence
	case SeekStart:
		offset += s.base // 从文件开头处开始seek，下次读写的偏移量等于偏移量+SectionReader本身开始位置
	case SeekCurrent:
		offset += s.off // 从当前位置开始seek， 下次读写的偏移量等于偏移量+SectionReader本身的偏移量
	case SeekEnd:
		offset += s.limit // 从文件末尾开始seek，下次读写的偏移量等于偏移量+SectionReader本身的limit数
	}
	if offset < s.base {
		return 0, errOffset // 非法的offset
	}
	s.off = offset              // 将s本身的offset设置为offset
	return offset - s.base, nil //
}

// 实现了ReadAt接口
func (s *SectionReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.limit-s.base {
		return 0, EOF
	}
	off += s.base
	if max := s.limit - off; int64(len(p)) > max {
		p = p[0:max]
		n, err = s.r.ReadAt(p, off)
		if err == nil {
			err = EOF
		}
		return n, err
	}
	return s.r.ReadAt(p, off)
}

// Size returns the size of the section in bytes.
func (s *SectionReader) Size() int64 { return s.limit - s.base }

// TeeReader returns a Reader that writes to w what it reads from r.
// All reads from r performed through it are matched with
// corresponding writes to w. There is no internal buffering -
// the write must complete before the read completes.
// Any error encountered while writing is reported as a read error.
// 类似于Unix中的tee命令， 从r中读取多少就立即写入w
func TeeReader(r Reader, w Writer) Reader {
	return &teeReader{r, w}
}

type teeReader struct {
	r Reader
	w Writer
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}
