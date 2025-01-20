package main

import (
	"fmt"
	"io"
	"sync"

	crunch "github.com/superwhiskers/crunch/v3"
)

type Buffer struct {
	sync.Mutex
	buffer *crunch.Buffer
	parent *Buffer
	length int64
	offset int64
	closed bool
	name   string
}

func NewBuffer(name string, slices ...[]byte) *Buffer {
	b := new(Buffer)
	b.buffer = crunch.NewBuffer(slices...)
	b.length = b.buffer.ByteCapacity()
	b.SetName(name)
	return b
}

func (b *Buffer) Read(dst []byte) (read int, err error) {
	if b == nil {
		panic("READ: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	if b.Closed() {
		return 0, io.EOF
	}
	if b.parent != nil {
		read, err = b.parent.ReadOffset(dst, b.offset)
		b.offset += int64(read)
		return
	}
	buffer := b.Buffer()
	if buffer == nil {
		return 0, fmt.Errorf("buffer: read: crunch buffer vanished")
	}
	if b.offset >= b.length {
		b.offset = b.length
	}
	toRead := b.length - b.offset
	if int(toRead) > len(dst) {
		toRead = int64(len(dst))
	}
	if toRead == 0 {
		return 0, nil
	}
	bytes := buffer.ReadBytes(b.offset, toRead)
	read = copy(dst, bytes)
	b.offset += int64(read)
	return
}

func (b *Buffer) ReadOffset(dst []byte, offset int64) (read int, err error) {
	if b == nil {
		panic("READOFFSET: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	if b.Closed() {
		return 0, io.EOF
	}
	buffer := b.Buffer()
	if buffer == nil {
		return 0, fmt.Errorf("buffer: readoffset: crunch buffer vanished")
	}
	toRead := b.length - offset
	if int(toRead) > len(dst) {
		toRead = int64(len(dst))
	}
	if toRead == 0 {
		return 0, nil
	}
	bytes := buffer.ReadBytes(offset, toRead)
	read = copy(dst, bytes)
	return
}

func (b *Buffer) Write(src []byte) (wrote int, err error) {
	if b == nil {
		panic("WRITE: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	if b.Closed() {
		return 0, io.EOF
	}
	if b.parent != nil {
		wrote, err = b.parent.WriteOffset(src, b.offset)
		b.offset += int64(wrote)
		return
	}
	buffer := b.Buffer()
	if buffer == nil {
		return 0, fmt.Errorf("buffer: write: crunch buffer vanished")
	}
	if toGrow := (b.offset + int64(len(src))) - b.length; toGrow > 0 {
		b.length += toGrow
		buffer.Grow(toGrow)
	}
	buffer.WriteBytes(b.offset, src)
	wrote = len(src)
	b.offset += int64(wrote)
	return
}

func (b *Buffer) WriteOffset(src []byte, offset int64) (wrote int, err error) {
	if b == nil {
		panic("WRITEOFFSET: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	if b.Closed() {
		return 0, io.EOF
	}
	buffer := b.Buffer()
	if buffer == nil {
		return 0, fmt.Errorf("buffer: writeoffset: crunch buffer vanished")
	}
	if toGrow := (offset + int64(len(src))) - b.length; toGrow > 0 {
		b.length += toGrow
		buffer.Grow(toGrow)
	}
	buffer.WriteBytes(offset, src)
	wrote = len(src)
	return
}

func (b *Buffer) Seek(to int64, whence int) (offset int64, err error) {
	if b == nil {
		panic("SEEK: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	if b.Closed() {
		return 0, io.EOF
	}
	buffer := b.Buffer()
	if buffer == nil {
		return 0, fmt.Errorf("buffer: seek: crunch buffer vanished")
	}
	switch whence {
	case io.SeekStart:
		b.offset = to
	case io.SeekCurrent:
		b.offset += to
	case io.SeekEnd:
		b.offset = b.length - to
	}
	offset = b.offset
	if b.parent == nil {
		buffer.SeekByte(offset, false)
	}
	return
}

func (b *Buffer) Close() error {
	if b == nil {
		panic("CLOSE: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	b.closed = true
	if b.parent != nil {
		return b.parent.Close()
	}
	return nil
}

func (b *Buffer) Closed() bool {
	if b == nil {
		panic("CLOSED: buffer is nil")
	}
	if b.parent != nil {
		return b.parent.Closed()
	}
	return b.closed
}

func (b *Buffer) Buffer() *crunch.Buffer {
	if b == nil {
		panic("BUFFER: buffer is nil")
	}
	buffer := b.buffer
	if b.parent != nil {
		buffer = b.parent.Buffer()
	}
	b.length = buffer.ByteCapacity()
	return buffer
}

func (b *Buffer) ByteCapacity() int64 {
	if b == nil {
		panic("BYTECAPACITY: buffer is nil")
	}
	if buffer := b.Buffer(); buffer != nil {
		return b.length
	}
	return 0
}

func (b *Buffer) Reference() *Buffer {
	if b == nil {
		panic("REFERENCE: buffer is nil")
	}
	nb := new(Buffer)
	nb.parent = b
	return nb
}

func (b *Buffer) Copy() *Buffer {
	if b == nil {
		panic("COPY: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	nb := new(Buffer)
	nb.buffer = crunch.NewBuffer(b.buffer.Bytes())
	nb.length = b.length
	return nb
}

func (b *Buffer) Bytes() []byte {
	if b == nil {
		panic("BYTES: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	return b.Buffer().Bytes()
}

func (b *Buffer) SetName(name string) {
	if b == nil {
		panic("SETNAME: buffer is nil")
	}
	b.name = name
}

func (b *Buffer) GetName() string {
	if b == nil {
		panic("GETNAME: buffer is nil")
	}
	return b.name
}
