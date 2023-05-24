package f_repository_impl

import "sync"

type LazyLoad[T any] struct {
	m      sync.Mutex
	done   bool
	value  T
	err    error
	loadFn func() (T, error)
}

func LazyLoadFn[T any](load func() (T, error)) *LazyLoad[T] {
	return &LazyLoad[T]{loadFn: load}
}
func LazyLoadValue[T any](v T) *LazyLoad[T] {
	return &LazyLoad[T]{
		done:  true,
		value: v,
	}
}

func (l *LazyLoad[T]) Get() T {
	l.m.Lock()
	defer l.m.Unlock()
	if l.done {
		return l.value
	}
	l.value, l.err = l.loadFn()
	l.done = true
	return l.value
}
