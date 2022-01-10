package queue

type Option func(*Queue)

func WithCapactity(c uint32) Option {
	return func(q *Queue) {
		q.capacity = c
	}
}
