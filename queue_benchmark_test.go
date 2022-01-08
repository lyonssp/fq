package queue

import (
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func benchmarkEnqueue(b *testing.B, value []byte) {
	f, err := ioutil.TempFile("", "test-*")
	assert := assert.New(b)
	assert.Nil(err)

	q := NewQueue(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		q.Enqueue(value)
	}
}

func BenchmarkEnqueue5(b *testing.B)   { benchmarkEnqueue(b, nBytes(5)) }
func BenchmarkEnqueue10(b *testing.B)  { benchmarkEnqueue(b, nBytes(10)) }
func BenchmarkEnqueue50(b *testing.B)  { benchmarkEnqueue(b, nBytes(50)) }
func BenchmarkEnqueue100(b *testing.B) { benchmarkEnqueue(b, nBytes(100)) }

func benchmarkDequeue(b *testing.B, value []byte) {
	f, err := ioutil.TempFile("", "test-*")
	assert := assert.New(b)
	assert.Nil(err)

	q := NewQueue(f)

	for n := 0; n < b.N; n++ {
		q.Enqueue(value)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		q.Dequeue()
	}
}

func BenchmarkDequeue5(b *testing.B)   { benchmarkEnqueue(b, nBytes(5)) }
func BenchmarkDequeue10(b *testing.B)  { benchmarkEnqueue(b, nBytes(10)) }
func BenchmarkDequeue50(b *testing.B)  { benchmarkEnqueue(b, nBytes(50)) }
func BenchmarkDequeue100(b *testing.B) { benchmarkEnqueue(b, nBytes(100)) }

func nBytes(n int) []byte {
	bs := make([]byte, n)
	rand.Read(bs)
	return bs
}
