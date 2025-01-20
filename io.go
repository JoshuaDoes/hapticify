package main

import (
	"fmt"
	"io"
	"sync"

	crunch "github.com/superwhiskers/crunch/v3"
)

// Bytes requires a type to be able to represent itself as a byte slice
type Bytes interface {
	Bytes() []byte
}

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

func (b *Buffer) WriteAbstract(data any) (wrote int, err error) {
	buffer := crunch.NewBuffer()

	switch data.(type) {
	case io.Reader:
		bytes, readErr := io.ReadAll(data.(io.Reader))
		if readErr != nil {
			err = readErr
			return
		}
		buffer.Grow(int64(len(bytes)))
		buffer.WriteBytes(0, bytes)
	case Bytes:
		bytes := data.(Bytes).Bytes()
		buffer.Grow(int64(len(bytes)))
		buffer.WriteBytes(0, bytes)
	case byte, bool, int, uint:
		buffer.Grow(1)
		buffer.WriteByte(0, data.(byte))
	case []byte, string:
		bytes := data.([]byte)
		buffer.Grow(int64(len(bytes)))
		buffer.WriteBytes(0, bytes)
	case []string:
		strings := data.([]string)
		for i := 0; i < len(strings); i++ {
			buffer.Grow(int64(len(strings[i])))
			buffer.WriteBytesNext([]byte(strings[i]))
		}
	case int16:
		buffer.Grow(2)
		buffer.WriteI16LE(0, []int16{data.(int16)})
	case []int16:
		numbers := data.([]int16)
		buffer.Grow(int64(2 * len(numbers)))
		buffer.WriteI16LE(0, numbers)
	case int32:
		buffer.Grow(4)
		buffer.WriteI32LE(0, []int32{data.(int32)})
	case []int32:
		numbers := data.([]int32)
		buffer.Grow(int64(4 * len(numbers)))
		buffer.WriteI32LE(0, numbers)
	case int64:
		buffer.Grow(8)
		buffer.WriteI64LE(0, []int64{data.(int64)})
	case []int64:
		numbers := data.([]int64)
		buffer.Grow(int64(8 * len(numbers)))
		buffer.WriteI64LE(0, numbers)
	case uint16:
		buffer.Grow(2)
		buffer.WriteU16LE(0, []uint16{data.(uint16)})
	case []uint16:
		numbers := data.([]uint16)
		buffer.Grow(int64(2 * len(numbers)))
		buffer.WriteU16LE(0, numbers)
	case uint32:
		buffer.Grow(4)
		buffer.WriteU32LE(0, []uint32{data.(uint32)})
	case []uint32:
		numbers := data.([]uint32)
		buffer.Grow(int64(4 * len(numbers)))
		buffer.WriteU32LE(0, numbers)
	case uint64:
		buffer.Grow(8)
		buffer.WriteU64LE(0, []uint64{data.(uint64)})
	case []uint64:
		numbers := data.([]uint64)
		buffer.Grow(int64(8 * len(numbers)))
		buffer.WriteU64LE(0, numbers)
	case float32:
		buffer.Grow(4)
		buffer.WriteF32LE(0, []float32{data.(float32)})
	case []float32:
		numbers := data.([]float32)
		buffer.Grow(int64(4 * len(numbers)))
		buffer.WriteF32LE(0, numbers)
	case float64:
		buffer.Grow(8)
		buffer.WriteF64LE(0, []float64{data.(float64)})
	case []float64:
		numbers := data.([]float64)
		buffer.Grow(int64(8 * len(numbers)))
		buffer.WriteF64LE(0, numbers)
	default:
		err = fmt.Errorf("buffer: Unsupported type for abstract write: %v", data)
		return
	}

	wrote, err = b.Write(buffer.Bytes())
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

func (b *Buffer) Reset() {
	if b == nil {
		panic("RESET: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	b.length = 0
	b.offset = 0
	if b.parent != nil {
		b.parent.Reset()
		return
	}
	b.buffer.Reset()
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

func (b *Buffer) Size() int {
	if b == nil {
		panic("SIZE: buffer is nil")
	}
	return int(b.ByteCapacity())
}

func (b *Buffer) Bytes() []byte {
	if b == nil {
		panic("BYTES: buffer is nil")
	}
	b.Lock()
	defer b.Unlock()
	return b.Buffer().Bytes()
}

func (b *Buffer) String() string {
	if b == nil {
		panic("STRING: buffer is nil")
	}
	return string(b.Bytes())
}
