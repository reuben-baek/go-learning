package data

import (
	"fmt"
	"reflect"
	"sync"
)

type Lazy[T any] interface {
	Get() T
}

type LazyLoad[T any] struct {
	m      sync.Mutex
	done   bool
	value  T
	err    error
	loadFn func() (any, error)
}

func LazyLoadFn[T any](load func() (any, error)) *LazyLoad[T] {
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
	if l.loadFn == nil {
		l.done = true
		return l.value
	}
	var value any
	var ok bool
	value, l.err = l.loadFn()
	if l.err != nil {
		l.done = true
		return l.value
	}
	l.value, ok = value.(T)
	if !ok {
		panic(fmt.Sprintf("LazyLoad got %s, not %s", reflect.TypeOf(value), reflect.TypeOf(l.value)))
	}
	l.done = true
	return l.value
}
