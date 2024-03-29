package queue

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	headerLength        uint32 = 16 // 16 bytes
	elementHeaderLength uint32 = 8  // 4 next pointer bytes + 4 size bytes
)

var (
	ErrQueueFull  = errors.New("queue is full")
	ErrQueueEmpty = errors.New("cannot dequeue from empty queue")
)

// Queue is a FIFO queue backed by a file
type Queue struct {
	rws    io.ReadWriteSeeker
	header fileHeader // cached file header
}

func NewQueue(f io.ReadWriteSeeker) *Queue {
	q := &Queue{rws: f}

	// initialize queue state
	if err := q.init(); err != nil {
		panic(err)
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

// Enqueue will add a value to the queue
//
// If there is inadequate space between the tail position and the
// nearest boundary, where the boundary is either the end of the file
// or the position of the head element
func (ls *Queue) Enqueue(v []byte) error {
	bytesNeeded := uint32(4 + len(v))
	if bytesNeeded > ls.header.fileLength {
		return errors.New("element is too large to enqueue")
	}

	// check for queue fullness and seek to the appropriate position
	// when we can accept a write
	//
	// queue is full if there is neither space at
	// the end of the buffer nor at the front of the buffer
	//
	// writes do not wrap around the end of the buffer
	// to avoid needing to write twice
	var writePosition int64
	if bytesNeeded <= ls.tailSpaceAvailable() {
		writePosition = int64(ls.header.tailPosition)
	} else if bytesNeeded <= ls.headSpaceAvailable() {
		writePosition = int64(headerLength)
	} else {
		return ErrQueueFull
	}

	if _, err := ls.rws.Seek(writePosition, io.SeekStart); err != nil {
		return err
	}

	// Write new queue element
	elem := make([]byte, bytesNeeded)
	binary.BigEndian.PutUint32(elem[:4], uint32(len(v)))
	copy(elem[4:], v)
	n, err := ls.rws.Write(elem)
	if err != nil {
		return err
	}

	// Update local file header
	ls.header.tailPosition += uint32(n)
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
		return nil, ErrQueueEmpty
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

func (ls *Queue) headSpaceAvailable() uint32 {
	if ls.header.tailPosition < ls.header.headPosition {
		return ls.header.headPosition - ls.header.tailPosition
	}
	return ls.header.headPosition - headerLength
}

func (ls *Queue) tailSpaceAvailable() uint32 {
	// if queue is wrapped around the end of the buffer
	if ls.header.tailPosition < ls.header.headPosition {
		return ls.header.headPosition - ls.header.tailPosition
	}
	return ls.header.fileLength - ls.header.tailPosition
}

func (ls *Queue) defaultFileHeader() fileHeader {
	return fileHeader{4096, 0, 16, 16}
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

func (ls *Queue) readElementHeader(pos uint32) (uint32, error) {
	if _, err := ls.rws.Seek(int64(pos), io.SeekStart); err != nil {
		return 0, err
	}
	var header [4]byte
	if _, err := ls.rws.Read(header[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(header[:]), nil
}

type fileHeader struct {
	fileLength   uint32 // total length of the buffer backing a queue
	queueSize    uint32 // total number of elements in a queue
	headPosition uint32 // offset at which the first-in element can be found
	tailPosition uint32 // offset at which the last-in  element can be found
}
