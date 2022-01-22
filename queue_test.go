package queue

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

func TestQueueProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSize = 1 // ensures minimum one element generated in random slices

	properties := gopter.NewProperties(parameters)

	properties.Property("first enqueued element is always the result of dequeue", prop.ForAll(
		func(ss []string) (bool, error) {
			f, err := ioutil.TempFile("", "test-*")
			if err != nil {
				return false, err
			}

			q := NewQueue(f)

			for _, s := range ss {
				if err := q.Enqueue([]byte(s)); err != nil {
					return false, err
				}
			}

			front, err := q.Dequeue()
			if err != nil {
				return false, err
			}

			if !bytes.Equal(front, []byte(ss[0])) {
				return false, nil
			}

			return true, nil
		},
		gen.SliceOf(gen.Identifier()),
	))

	properties.Property("repeated enqueue and dequeue works", prop.ForAll(
		func(ss []string) (bool, error) {
			f, err := ioutil.TempFile("", "test-*")
			if err != nil {
				return false, err
			}

			q := NewQueue(f)

			for _, s := range ss {
				if err := q.Enqueue([]byte(s)); err != nil {
					return false, err
				}

				front, err := q.Dequeue()
				if err != nil {
					return false, err
				}

				if !bytes.Equal(front, []byte(s)) {
					return false, nil
				}
			}

			return true, nil
		},
		gen.SliceOf(gen.Identifier()),
	))

	properties.Property("file size never exceeds capacity", func(params *gopter.GenParameters) *gopter.PropResult {
		f, err := ioutil.TempFile("", "test-*")
		if err != nil {
			return &gopter.PropResult{Status: gopter.PropError, Error: err}
		}

		q := NewQueue(f)

		for i := 0; i < 10; i++ {
			cmd := genEnqueueDequeue(params).Result.(interface{})

			switch command := cmd.(type) {
			case enqueueCommand:
				err := q.Enqueue(command.x)
				if err == ErrQueueFull {
					return &gopter.PropResult{Status: gopter.PropUndecided}
				}
				if err != nil {
					return &gopter.PropResult{Status: gopter.PropError, Error: err}
				}
			case dequeueCommand:
				_, err := q.Dequeue()
				if err == ErrQueueEmpty {
					return &gopter.PropResult{Status: gopter.PropUndecided}
				}
				if err != nil {
					return &gopter.PropResult{Status: gopter.PropError, Error: err}
				}
			}
		}

		fi, err := f.Stat()
		if err != nil {
			return &gopter.PropResult{Status: gopter.PropError, Error: err}
		}

		if fi.Size() > int64(q.header.fileLength) {
			return gopter.NewPropResult(false, "file size is over capacity")
		}

		return gopter.NewPropResult(true, "")
	})

	properties.TestingRun(t)
}

// Capture failed model test sequences
func TestRegressions(t *testing.T) {
	assert := assert.New(t)

	t.Run("regression 0", func(t *testing.T) {
		f, err := ioutil.TempFile("", "test-*")
		assert.Nil(err)

		q := NewQueue(f)

		q.Enqueue([]byte("cz9qanCc"))
		q.Enqueue([]byte("wiekc00p"))
		q.Dequeue()
		q.Enqueue([]byte("t"))
		q.Dequeue()
		q.Enqueue([]byte("t"))
		q.Enqueue([]byte("h1lvfxhb"))
		check, err := q.Dequeue()
		assert.NotNil(check)

		front, err := q.Dequeue()
		assert.Nil(err)
		assert.Equal([]byte("t"), front)
	})

	t.Run("regression 1", func(t *testing.T) {
		f, err := ioutil.TempFile("", "test-*")
		assert.Nil(err)

		q := NewQueue(f)

		q.Enqueue([]byte("a"))
		q.Dequeue()
		q.Enqueue([]byte("b"))

		front, err := q.Dequeue()
		assert.Nil(err)
		assert.Equal([]byte("b"), front)
	})
}

// generate one of either an enqueueCommand or dequeueCommand at random
func genEnqueueDequeue(params *gopter.GenParameters) *gopter.GenResult {
	genEnqueue := func(p *gopter.GenParameters) *gopter.GenResult {
		val := gen.Identifier()(p).Result.(string)
		return gopter.NewGenResult(enqueueCommand{[]byte(val)}, gopter.NoShrinker)
	}

	genDequeue := func(p *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(dequeueCommand{}, gopter.NoShrinker)
	}

	if params.Rng.Intn(100)%2 == 0 {
		return genEnqueue(params)
	}

	return genDequeue(params)
}
