// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bufio implements buffered I/O. It wraps an io.Reader or io.Writer
// object, creating another object (Reader or Writer) that also implements
// the interface but provides buffering and some help for textual I/O.
// 带缓存的io包
package bufio

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

const (
	defaultBufSize = 4096 // 默认buffer大小： 4K
)

var (
	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
	ErrBufferFull        = errors.New("bufio: buffer full")
	ErrNegativeCount     = errors.New("bufio: negative count")
)

// Buffered input.

// Reader implements buffering for an io.Reader object.
type Reader struct {
	buf  []byte    // 缓存
	rd   io.Reader // reader provided by the client 底层的io.Reader
	r, w int       // buf read and write positions
	// r是从buf中读取走的字节偏移量， w是向buf中填充的字节偏移量
	// w - r是buf中可以读取的字节长度（缓存数据大小），同时也是b.Buffered()的返回值
	err          error
	lastByte     int // 最后一次读到的字节大小
	lastRuneSize int
}

const minReadBufferSize = 16
const maxConsecutiveEmptyReads = 100

// NewReaderSize returns a new Reader whose buffer has at least the specified
// size. If the argument io.Reader is already a Reader with large enough
// size, it returns the underlying Reader.
// 返回一个指定size的Reader实例，如果参数rd是一个足够大的Reader对象，那么就返回这个rd底层的Reader
func NewReaderSize(rd io.Reader, size int) *Reader {
	// Is it already a Reader?
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		return b // 直接返回底层Reader对象
	}
	if size < minReadBufferSize {
		size = minReadBufferSize // 最小读取大小为16字节
	}
	r := new(Reader)
	r.reset(make([]byte, size), rd) // 重置Reader
	return r
}

// NewReader returns a new Reader whose buffer has the default size.
// 返回一个默认大小的Reader实例
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

// Size returns the size of the underlying buffer in bytes.
// 返回该Reader实例中底层buf的大小
func (r *Reader) Size() int { return len(r.buf) }

// Reset discards any buffered data, resets all state, and switches
// the buffered reader to read from r.
// 重置整个Reader
func (b *Reader) Reset(r io.Reader) {
	b.reset(b.buf, r)
}

func (b *Reader) reset(buf []byte, r io.Reader) {
	// 将b底层的值重置
	*b = Reader{
		buf:          buf,
		rd:           r,
		lastByte:     -1,
		lastRuneSize: -1,
	}
}

var errNegativeRead = errors.New("bufio: reader returned negative count from Read")

// fill reads a new chunk into the buffer.
// 将新读取的块填充进buffer
func (b *Reader) fill() {
	// Slide existing data to beginning.
	// 将已经存在的数据滑动到开始处
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
	// 尝试从rd中读取数据到b.buf中，如果读取成功就返回，反之重试100次
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	// 如果处理到这还没有return，则将b.err填充为ErrNoProgress
	b.err = io.ErrNoProgress
}

// 将Reader中的错误提取出来，并将b.err重置为nil
func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

// Peek returns the next n bytes without advancing the reader. The bytes stop
// being valid at the next read call. If Peek returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than b's buffer size.
// 取出下n个字节，但是reader不会增长，Peek有偷看一下的意思
func (b *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}

	for b.w-b.r < n && b.w-b.r < len(b.buf) && b.err == nil {
		b.fill() // b.w-b.r < len(b.buf) => buffer is not full
	}

	if n > len(b.buf) {
		return b.buf[b.r:b.w], ErrBufferFull
	}

	// 0 <= n <= len(b.buf)
	var err error
	if avail := b.w - b.r; avail < n {
		// not enough data in buffer
		n = avail
		err = b.readErr()
		if err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err
}

// Discard skips the next n bytes, returning the number of bytes discarded.
//
// If Discard skips fewer than n bytes, it also returns an error.
// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
// reading from the underlying io.Reader.
// 跳过下n个字节， 并返回丢弃的字节数
func (b *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}
	remain := n // 将剩余字节数等于需要跳过的字节数
	for {
		skip := b.Buffered()
		if skip == 0 {
			b.fill() // 如果buffer中没有数据，填充一部分进来
			skip = b.Buffered()
		}
		if skip > remain {
			skip = remain // 如果已经存在（填充进）的字节数大于remain数，则将skip设置为与remain相等
		}
		b.r += skip      // 已经读取的字节数加上即将要读取的字节数
		remain -= skip   // 将remain数减去skip数
		if remain == 0 { // 当剩余字节数为0时，跳出循环，并返回已经读取的字节数
			return n, nil
		}
		if b.err != nil {
			return n - remain, b.readErr() // 这里只可能在fill()过程中出错，如果出错，返回相应的错误
		}
	}
}

// Read reads data into p.
// It returns the number of bytes read into p.
// The bytes are taken from at most one Read on the underlying Reader,
// hence n may be less than len(p).
// At EOF, the count will be zero and err will be io.EOF.
func (b *Reader) Read(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return 0, b.readErr()
	}
	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		if len(p) >= len(b.buf) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			// 读取量超过buffer的大小，需要直接从底层reader里读取，避免发生拷贝
			n, b.err = b.rd.Read(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			if n > 0 {
				b.lastByte = int(p[n-1]) // 更新最后读取的字节数
				b.lastRuneSize = -1      // 重置lastRuneSize
			}
			return n, b.readErr()
		}
		// One read.
		// Do not use b.fill, which will loop.
		b.r = 0
		b.w = 0                     // 重置b.r 和 b.w ，这里不是线程安全的
		n, b.err = b.rd.Read(b.buf) // 先从底层reader中读取到b.buf中
		if n < 0 {
			panic(errNegativeRead)
		}
		if n == 0 {
			return 0, b.readErr()
		}
		b.w += n // 填充进(写入)buffer完成，增加b.w
	}

	// copy as much as we can
	// 尽可能地拷贝
	n = copy(p, b.buf[b.r:b.w])
	b.r += n                       // 拷贝(读取)buffer完成，增加b.r
	b.lastByte = int(b.buf[b.r-1]) // 更新lastByte
	b.lastRuneSize = -1            // 重置lastRuneSize
	return n, nil
}

// ReadByte reads and returns a single byte.
// If no byte is available, returns an error.
// 读取并返回单一字节， 如果没有可用字节，则返回有一个错误
func (b *Reader) ReadByte() (byte, error) {
	b.lastRuneSize = -1 // 重置lastRuneSize
	for b.r == b.w {    // b.r == b.w => buffer is empty
		if b.err != nil {
			return 0, b.readErr()
		}
		// 填充buffer
		b.fill() // buffer is empty
	}
	c := b.buf[b.r]     // 从b.r处向后读取一个字节
	b.r++               // b.r自增1
	b.lastByte = int(c) // 更新lastByte
	return c, nil       // 返回读取的字节
}

// UnreadByte unreads the last byte. Only the most recently read byte can be unread.
// 将上次ReadByte的字节还原
func (b *Reader) UnreadByte() error {
	if b.lastByte < 0 || b.r == 0 && b.w > 0 { // 最后读取字节数小于0 或者 直接没有读取且buffer不为空
		return ErrInvalidUnreadByte
	}
	// b.r > 0 || b.w == 0
	if b.r > 0 {
		b.r-- // b.r递减1
	} else {
		// b.r == 0 && b.w == 0
		b.w = 1 // 写入一个字节，更新b.w
	}
	b.buf[b.r] = byte(b.lastByte)
	b.lastByte = -1
	b.lastRuneSize = -1 // 重置lastByte和lastRuneSize
	return nil
}

// ReadRune reads a single UTF-8 encoded Unicode character and returns the
// rune and its size in bytes. If the encoded rune is invalid, it consumes one byte
// and returns unicode.ReplacementChar (U+FFFD) with a size of 1.
// 读取单个UTF-8 Unicode代码点，并返回它的大小（以bytes为维度）
func (b *Reader) ReadRune() (r rune, size int, err error) {
	for b.r+utf8.UTFMax > b.w && !utf8.FullRune(b.buf[b.r:b.w]) && b.err == nil && b.w-b.r < len(b.buf) {
		b.fill() // b.w-b.r < len(buf) => buffer is not full
	}
	b.lastRuneSize = -1
	if b.r == b.w {
		return 0, 0, b.readErr()
	}
	r, size = rune(b.buf[b.r]), 1
	if r >= utf8.RuneSelf {
		r, size = utf8.DecodeRune(b.buf[b.r:b.w])
	}
	b.r += size
	b.lastByte = int(b.buf[b.r-1])
	b.lastRuneSize = size
	return r, size, nil
}

// UnreadRune unreads the last rune. If the most recent read operation on
// the buffer was not a ReadRune, UnreadRune returns an error.  (In this
// regard it is stricter than UnreadByte, which will unread the last byte
// from any read operation.)
// 将上次ReadRune调用中读取的Rune还原
func (b *Reader) UnreadRune() error {
	if b.lastRuneSize < 0 || b.r < b.lastRuneSize {
		return ErrInvalidUnreadRune
	}
	b.r -= b.lastRuneSize
	b.lastByte = -1
	b.lastRuneSize = -1
	return nil
}

// Buffered returns the number of bytes that can be read from the current buffer.
// 返回当前buffer中可以读取的字节数
func (b *Reader) Buffered() int { return b.w - b.r }

// ReadSlice reads until the first occurrence of delim in the input,
// returning a slice pointing at the bytes in the buffer.
// The bytes stop being valid at the next read.
// If ReadSlice encounters an error before finding a delimiter,
// it returns all the data in the buffer and the error itself (often io.EOF).
// ReadSlice fails with error ErrBufferFull if the buffer fills without a delim.
// Because the data returned from ReadSlice will be overwritten
// by the next I/O operation, most clients should use
// ReadBytes or ReadString instead.
// ReadSlice returns err != nil if and only if line does not end in delim.
// ReadSlice从输入中读取，直到遇到第一个界定符（delim）为止，返回一个指向缓存中字节的slice，在下次调用读操作（read）时，这些字节会无效
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	for {
		// Search buffer.
		if i := bytes.IndexByte(b.buf[b.r:b.w], delim); i >= 0 {
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		// Pending error?
		if b.err != nil {
			line = b.buf[b.r:b.w] // line 指向了buffer
			b.r = b.w
			err = b.readErr()
			break
		}

		// Buffer full?
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}

		b.fill() // buffer is not full
	}

	// Handle last byte, if any.
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
		b.lastRuneSize = -1
	}

	return
}

// ReadLine is a low-level line-reading primitive. Most callers should use
// ReadBytes('\n') or ReadString('\n') instead or use a Scanner.
//
// ReadLine tries to return a single line, not including the end-of-line bytes.
// If the line was too long for the buffer then isPrefix is set and the
// beginning of the line is returned. The rest of the line will be returned
// from future calls. isPrefix will be false when returning the last fragment
// of the line. The returned buffer is only valid until the next call to
// ReadLine. ReadLine either returns a non-nil line or it returns an error,
// never both.
//
// The text returned from ReadLine does not include the line end ("\r\n" or "\n").
// No indication or error is given if the input ends without a final line end.
// Calling UnreadByte after ReadLine will always unread the last byte read
// (possibly a character belonging to the line end) even if that byte is not
// part of the line returned by ReadLine.
// ReadLine是一个底层的原始行读取命令。许多调用者或许会使用ReadBytes('\n')或者ReadString('\n')来代替这个方法。
// ReadLine尝试返回单独的行，不包括行尾的换行符。如果一行大于缓存，isPrefix会被设置为true，同时返回该行的开始部分（等于缓存大小的部分）。
// 该行剩余的部分就会在下次调用的时候返回。当下次调用返回该行剩余部分时，isPrefix将会是false。
// 跟ReadSlice一样，返回的line只是buffer的引用，在下次执行IO操作时，line会无效。可以将ReadSlice中的例子该为ReadLine试试。
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	line, err = b.ReadSlice('\n') // 通过换行符先读取一行，这里有读取的行可能不完整，超过了单行超过了buffer大小
	if err == ErrBufferFull {
		// Handle the case where "\r\n" straddles the buffer.
		// 处理"\r\n"横跨buffer的情况
		if len(line) > 0 && line[len(line)-1] == '\r' { // 之前line已经读取到了部分数据，如果line最后一个字符是'\r'的话
			// Put the '\r' back on buf and drop it from line.
			// Let the next call to ReadLine check for "\r\n".
			// 将'\r'放回到buffer中，等待下次再检查'\r\n'
			if b.r == 0 { // 这里不可能发生
				// should be unreachable
				panic("bufio: tried to rewind past start of buffer")
			}
			b.r--                     // r递减1
			line = line[:len(line)-1] // 删掉最后一个'\r'
		}
		return line, true, nil
	}

	if len(line) == 0 {
		if err != nil {
			line = nil // 如果出错，返回空line
		}
		return
	}
	err = nil // 到这里，表示读取line正常

	if line[len(line)-1] == '\n' { // 丢弃line中结尾处的'\n'
		drop := 1
		if len(line) > 1 && line[len(line)-2] == '\r' { // 如果结尾处有'\r'也要丢弃掉
			drop = 2
		}
		line = line[:len(line)-drop]
	}
	return
}

// ReadBytes reads until the first occurrence of delim in the input,
// returning a slice containing the data up to and including the delimiter.
// If ReadBytes encounters an error before finding a delimiter,
// it returns the data read before the error and the error itself (often io.EOF).
// ReadBytes returns err != nil if and only if the returned data does not end in
// delim.
// For simple uses, a Scanner may be more convenient.
// ReadBytes从输入中读取，直到遇到第一个界定符（delim）为止，返回一个包含这个界定符的切片(拷贝)
func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	// Use ReadSlice to look for array,
	// accumulating full buffers.
	var frag []byte
	var full [][]byte
	var err error
	for {
		var e error
		frag, e = b.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != ErrBufferFull { // unexpected error
			err = e
			break
		}

		// Make a copy of the buffer.
		buf := make([]byte, len(frag))
		copy(buf, frag)
		full = append(full, buf)
	}

	// Allocate new buffer to hold the full pieces and the fragment.
	n := 0
	for i := range full {
		n += len(full[i])
	}
	n += len(frag)

	// Copy full pieces and fragment in.
	buf := make([]byte, n)
	n = 0
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)
	return buf, err
}

// ReadString reads until the first occurrence of delim in the input,
// returning a string containing the data up to and including the delimiter.
// If ReadString encounters an error before finding a delimiter,
// it returns the data read before the error and the error itself (often io.EOF).
// ReadString returns err != nil if and only if the returned data does not end in
// delim.
// For simple uses, a Scanner may be more convenient.
// 同ReadBytes()，返回string， 这里会做一个[]byte到string的转换
func (b *Reader) ReadString(delim byte) (string, error) {
	bytes, err := b.ReadBytes(delim)
	return string(bytes), err
}

// WriteTo implements io.WriterTo.
// This may make multiple calls to the Read method of the underlying Reader.
// If the underlying reader supports the WriteTo method,
// this calls the underlying WriteTo without buffering.
// 实现了io.WriterTo接口
// 这个方法会多次调用底层Reader的Read方法，如果底层的Reader支持WriteTo方法，那么就无须调用底层buffer的WriteTo方法
func (b *Reader) WriteTo(w io.Writer) (n int64, err error) {
	n, err = b.writeBuf(w) // 将底层Reader的buffer写入到w中
	if err != nil {
		return
	}

	if r, ok := b.rd.(io.WriterTo); ok { // 检测底层的Reader是否实现了io.WriterTo接口
		m, err := r.WriteTo(w) // 直接调用底层Reader的WriteTo()方法
		n += m
		return n, err
	}

	if w, ok := w.(io.ReaderFrom); ok { // 检测底层的Reader是否实现了io.ReaderFrom接口
		m, err := w.ReadFrom(b.rd) // 直接调用底层Reader的ReadFrom()方法
		n += m
		return n, err
	}

	if b.w-b.r < len(b.buf) {
		b.fill() // buffer not full
	}

	for b.r < b.w {
		// b.r < b.w => buffer is not empty
		m, err := b.writeBuf(w)
		n += m
		if err != nil {
			return n, err
		}
		b.fill() // buffer is empty
	}

	if b.err == io.EOF {
		b.err = nil
	}

	return n, b.readErr()
}

var errNegativeWrite = errors.New("bufio: writer returned negative count from Write")

// writeBuf writes the Reader's buffer to the writer.
// 将底层的Reader的buffer写入到w中
func (b *Reader) writeBuf(w io.Writer) (int64, error) {
	n, err := w.Write(b.buf[b.r:b.w])
	if n < 0 {
		panic(errNegativeWrite)
	}
	b.r += n
	return int64(n), err
}

// buffered output

// Writer implements buffering for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes, and Flush, will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
// Writer结构实现了带buffer的io.Writer接口对象
// 当写入到底层Writer时发生错误，后续的数据都将会停止写入，之前写入的数据将会被刷盘到底层writer中并返回错误。
// 当所有的写入结束时，调用方需要显式调用Flush()方法来保证所有的数据都已经转发到底层的io.Writer中
type Writer struct {
	err error     // 写入过程中遇到的错误
	buf []byte    // 底层buffer
	n   int       // 写入的字节数
	wr  io.Writer // 底层io.Writer
}

// NewWriterSize returns a new Writer whose buffer has at least the specified
// size. If the argument io.Writer is already a Writer with large enough
// size, it returns the underlying Writer.
// 返回一个buffer至少为size字节的带缓冲的Writer，如果参数中的w已经是一个Writer接口对象(就是bufio.Writer本身)，并且大小足够，
// 就直接返回底层的Writer
func NewWriterSize(w io.Writer, size int) *Writer {
	// Is it already a Writer?
	b, ok := w.(*Writer)
	if ok && len(b.buf) >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufSize // 4Kb
	}
	return &Writer{
		buf: make([]byte, size),
		wr:  w, // err == nil, n == 0,
	}
}

// NewWriter returns a new Writer whose buffer has the default size.
// 返回一个默认大小buffer的Writer
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

// Size returns the size of the underlying buffer in bytes.
// 返回当前buffer的大小
func (b *Writer) Size() int { return len(b.buf) }

// Reset discards any unflushed buffered data, clears any error, and
// resets b to write its output to w.
// 清除当前未刷写的buffer，清除错误，并重置底层writer
func (b *Writer) Reset(w io.Writer) {
	b.err = nil
	b.n = 0
	b.wr = w
	// 为什么不将buf也重置掉？避免分配？
	// b.buf = make([]byte, 0)
}

// Flush writes any buffered data to the underlying io.Writer.
// 将buffer中的数据刷写到底层的底层的io.Writer接口对象中
func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err // 如果之前的写入过程中出现了错误，就停止刷写，并返回写入时的错误
	}
	if b.n == 0 { // n表示当前已经写入的字节数，0表示没有任何写入，直接返回nil
		return nil
	}
	n, err := b.wr.Write(b.buf[0:b.n]) // 调用底层io.Writer接口对象的Write()方法将已经写入的字节数flush到b.wr中
	if n < b.n && err == nil {         // 未能完全刷入，但未发生错误，返回预定义的错误
		err = io.ErrShortWrite
	}
	if err != nil { // 刷入时发生了一个错误
		if n > 0 && n < b.n { // 因为发生了一个错误，而导致未能完全刷入
			copy(b.buf[0:b.n-n], b.buf[n:b.n]) // 回滚之前的刷入操作，即将已经刷入的数据再copy回底层的buffer中
		}
		b.n -= n    // 更新b.n
		b.err = err // 更新err
		return err  // 返回err
	}
	b.n = 0 // 输入成功，重置b.n 并返回nil
	return nil
}

// Available returns how many bytes are unused in the buffer.
// 返回当前底层buffer中有多少可用的字节数
func (b *Writer) Available() int { return len(b.buf) - b.n }

// Buffered returns the number of bytes that have been written into the current buffer.
// 返回当前底层buffer中已经完成写入（不是刷入：flush）的字节数
func (b *Writer) Buffered() int { return b.n }

// Write writes the contents of p into the buffer.
// It returns the number of bytes written.
// If nn < len(p), it also returns an error explaining
// why the write is short.
// 将p的内容写入到底层buffer中， 并返回写入量nn
// 如果nn < len(p)， 则是部分写入，同样也返回一个明确的错误
func (b *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			n, b.err = b.wr.Write(p)
		} else {
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.Flush()
		}
		nn += n
		p = p[n:]
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	nn += n
	return nn, nil
}

// WriteByte writes a single byte.
func (b *Writer) WriteByte(c byte) error {
	if b.err != nil {
		return b.err
	}
	if b.Available() <= 0 && b.Flush() != nil {
		return b.err
	}
	b.buf[b.n] = c
	b.n++
	return nil
}

// WriteRune writes a single Unicode code point, returning
// the number of bytes written and any error.
func (b *Writer) WriteRune(r rune) (size int, err error) {
	if r < utf8.RuneSelf {
		err = b.WriteByte(byte(r))
		if err != nil {
			return 0, err
		}
		return 1, nil
	}
	if b.err != nil {
		return 0, b.err
	}
	n := b.Available()
	if n < utf8.UTFMax {
		if b.Flush(); b.err != nil {
			return 0, b.err
		}
		n = b.Available()
		if n < utf8.UTFMax {
			// Can only happen if buffer is silly small.
			return b.WriteString(string(r))
		}
	}
	size = utf8.EncodeRune(b.buf[b.n:], r)
	b.n += size
	return size, nil
}

// WriteString writes a string.
// It returns the number of bytes written.
// If the count is less than len(s), it also returns an error explaining
// why the write is short.
func (b *Writer) WriteString(s string) (int, error) {
	nn := 0
	for len(s) > b.Available() && b.err == nil {
		n := copy(b.buf[b.n:], s)
		b.n += n
		nn += n
		s = s[n:]
		b.Flush()
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], s)
	b.n += n
	nn += n
	return nn, nil
}

// ReadFrom implements io.ReaderFrom. If the underlying writer
// supports the ReadFrom method, and b has no buffered data yet,
// this calls the underlying ReadFrom without buffering.
func (b *Writer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.Buffered() == 0 {
		if w, ok := b.wr.(io.ReaderFrom); ok {
			return w.ReadFrom(r)
		}
	}
	var m int
	for {
		if b.Available() == 0 {
			if err1 := b.Flush(); err1 != nil {
				return n, err1
			}
		}
		nr := 0
		for nr < maxConsecutiveEmptyReads {
			m, err = r.Read(b.buf[b.n:])
			if m != 0 || err != nil {
				break
			}
			nr++
		}
		if nr == maxConsecutiveEmptyReads {
			return n, io.ErrNoProgress
		}
		b.n += m
		n += int64(m)
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		// If we filled the buffer exactly, flush preemptively.
		if b.Available() == 0 {
			err = b.Flush()
		} else {
			err = nil
		}
	}
	return n, err
}

// buffered input and output

// ReadWriter stores pointers to a Reader and a Writer.
// It implements io.ReadWriter.
type ReadWriter struct {
	*Reader
	*Writer
}

// NewReadWriter allocates a new ReadWriter that dispatches to r and w.
func NewReadWriter(r *Reader, w *Writer) *ReadWriter {
	return &ReadWriter{r, w}
}
