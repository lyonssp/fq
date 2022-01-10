package queue

import (
	"encoding/binary"
	"errors"
	"io"
)

const defaultQueueCapacity = 4096

// Queue is a FIFO queue backed by a file
type Queue struct {
	rws    io.ReadWriteSeeker
	header fileHeader // cached file header

	capacity uint32
}

func NewQueue(f io.ReadWriteSeeker, opts ...Option) *Queue {
	q := &Queue{rws: f, capacity: defaultQueueCapacity}
	if err := q.init(); err != nil {
		panic(err)
	}

	for _, opt := range opts {
		opt(q)
	}

	return q
}

// init will initialize Queue.rws and load any requisite in-memory state
func (ls *Queue) init() error {
	ls.header = ls.defaultFileHeader()

	header, err := ls.readHeader()
	if err == io.EOF {
		// if here we are initializing for the first time
		// and need to write the default header
		return ls.syncHeader()
	}

	if err != nil {
		return err
	}

	ls.header = header
	return nil
}

// syncHeader writes the in-memory queue header to Queue.rws
func (ls *Queue) syncHeader() error {
	// Build header buffer
	var headerBytes [16]byte
	binary.BigEndian.PutUint32(headerBytes[:4], ls.header.fileLength)
	binary.BigEndian.PutUint32(headerBytes[4:8], ls.header.queueSize)
	binary.BigEndian.PutUint32(headerBytes[8:12], ls.header.headPosition)
	binary.BigEndian.PutUint32(headerBytes[12:], ls.header.tailPosition)

	// Write header
	if _, err := ls.rws.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := ls.rws.Write(headerBytes[:]); err != nil {
		return err
	}

	return nil
}

// Enqueue the value x to the back of the queue
func (ls *Queue) Enqueue(v []byte) error {

	// Write new queue element to the tail pointer
	elem := make([]byte, 4+len(v))
	binary.BigEndian.PutUint32(elem[:4], uint32(len(v)))
	copy(elem[4:], v)

	if _, err := ls.rws.Seek(int64(ls.header.tailPosition), io.SeekStart); err != nil {
		return err
	}
	n, err := ls.rws.Write(elem)
	if err != nil {
		return err
	}

	// Update local file header
	//
	// tail position only gets updated from the default when enqueueing into a non-empty queue
	ls.header.tailPosition = ls.header.tailPosition + uint32(n)
	ls.header.queueSize += 1

	// Sync header updates to finalize the write
	if err := ls.syncHeader(); err != nil {
		return err
	}

	return nil
}

// Dequeue and return the item at the front of the queue
func (ls *Queue) Dequeue() ([]byte, error) {
	if ls.header.queueSize == 0 {
		return nil, errors.New("cannot dequeue from empty queue")
	}

	// Seek to first element
	if _, err := ls.rws.Seek(int64(ls.header.headPosition), io.SeekStart); err != nil {
		return nil, err
	}

	// Read element length from its header
	var elementHeader [4]byte
	if _, err := ls.rws.Read(elementHeader[:]); err != nil {
		return nil, err
	}

	// Read element data
	elementLength := binary.BigEndian.Uint32(elementHeader[:])
	elementData := make([]byte, elementLength)
	if _, err := ls.rws.Read(elementData[:]); err != nil {
		return nil, err
	}

	ls.header.headPosition += elementLength + 4 // head position moves the length of the removed element plus its header
	ls.header.queueSize -= 1

	if ls.header.queueSize == 0 {
		ls.header = ls.defaultFileHeader()
	}

	// Sync header updates to finalize the write
	if err := ls.syncHeader(); err != nil {
		return nil, err
	}

	return elementData, nil
}

func (ls *Queue) defaultFileHeader() fileHeader {
	return fileHeader{ls.capacity, 0, 16, 16}
}

func (ls *Queue) readHeader() (fileHeader, error) {
	if _, err := ls.rws.Seek(0, io.SeekStart); err != nil {
		return fileHeader{}, err
	}

	var headerBytes [16]byte
	if _, err := io.ReadFull(ls.rws, headerBytes[:]); err != nil {
		return fileHeader{}, err
	}

	return fileHeader{
		fileLength:   binary.BigEndian.Uint32(headerBytes[:4]),
		queueSize:    binary.BigEndian.Uint32(headerBytes[4:8]),
		headPosition: binary.BigEndian.Uint32(headerBytes[8:12]),
		tailPosition: binary.BigEndian.Uint32(headerBytes[12:]),
	}, nil
}

type fileHeader struct {
	fileLength   uint32 // total length of the buffer backing a queue
	queueSize    uint32 // total number of elements in a queue
	headPosition uint32 // offset at which the first-in element can be found
	tailPosition uint32 // offset at which the last-in  element can be found
}

func (header fileHeader) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint32(b[:4], header.fileLength)
	binary.BigEndian.PutUint32(b[4:8], header.queueSize)
	binary.BigEndian.PutUint32(b[8:12], header.headPosition)
	binary.BigEndian.PutUint32(b[12:], header.tailPosition)
	return b, nil
}

func (header *fileHeader) UnmarshalBinary(bs []byte) error {
	header.fileLength = binary.BigEndian.Uint32(bs[:4])
	header.queueSize = binary.BigEndian.Uint32(bs[4:8])
	header.headPosition = binary.BigEndian.Uint32(bs[8:12])
	header.tailPosition = binary.BigEndian.Uint32(bs[12:])
	return nil
}
