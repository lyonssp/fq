package queue

type Option func(*Queue)

func WithCapacity(c uint32) Option {
	return func(q *Queue) {
		q.capacity = c
	}
}
