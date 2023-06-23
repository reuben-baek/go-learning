package data

import (
	"fmt"
	"github.com/sirupsen/logrus"
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

type LazyLoadable interface {
	NewInstance()
	SetLoadFunc(name string, fn func() (any, error))
	HasLoadFunc(name string) bool
	DeleteLoadFunc(name string)
	Load(name string, emptyEntity any) (any, error) // Load returns entity
	Entities() []string
}

type LazyLoader struct {
	loaderMap map[string]func() (any, error)
}

func (l *LazyLoader) NewInstance() {
	l.loaderMap = make(map[string]func() (any, error))
}

func (l *LazyLoader) SetLoadFunc(entity string, fn func() (any, error)) {
	l.loaderMap[entity] = fn
}

func (l *LazyLoader) DeleteLoadFunc(entity string) {
	delete(l.loaderMap, entity)
}

func (l *LazyLoader) Entities() []string {
	entities := make([]string, 0, len(l.loaderMap))
	for k, _ := range l.loaderMap {
		entities = append(entities, k)
	}
	return entities
}

func (l *LazyLoader) HasLoadFunc(entity string) bool {
	_, ok := l.loaderMap[entity]
	return ok
}

func (l *LazyLoader) Load(name string, emptyEntity any) (any, error) {
	typeOf := reflect.TypeOf(emptyEntity)
	if fn, ok := l.loaderMap[name]; ok {
		loadedEntity, err := fn()
		logrus.Debugf("LazyLoader.Load: LazyLoader[%p] loaded [%s] [%+v], err[%v]", l, typeOf.String(), loadedEntity, err)
		delete(l.loaderMap, name)
		return loadedEntity, err
	} else {
		return nil, fmt.Errorf("lazy load function for %s[%s] is not set", name, typeOf)
	}
}

func LazyLoadNow[T any](name string, lazyLoader LazyLoadable) (T, error) {
	var entity T
	var err error
	var loaded any
	loaded, err = lazyLoader.Load(name, entity)
	if loaded == nil || err != nil {
		return entity, err
	}
	valueOfParent := reflect.ValueOf(lazyLoader)
	valueOfLoaded := reflect.ValueOf(loaded)

	child := reflect.Indirect(valueOfParent).FieldByName(name)
	if child.Type().Kind() == reflect.Pointer {
		logrus.Debugf("LazyLoadNow: parent[%s] field[%s %s] value[%s %s]", valueOfParent.Type().String(), name, child.Type().String(), valueOfLoaded.Type().String(), valueOfLoaded.Interface())
		child.Set(reflect.New(reflect.TypeOf(loaded)))
		child.Elem().Set(valueOfLoaded)
		return child.Interface().(T), err
	} else {
		logrus.Debugf("LazyLoadNow: parent[%s] field[%s %s] value[%s %s]", valueOfParent.Type().String(), name, child.Type().String(), valueOfLoaded.Type().String(), valueOfLoaded.Interface())
		child.Set(reflect.Indirect(valueOfLoaded))
		return reflect.Indirect(reflect.ValueOf(loaded)).Interface().(T), err
	}
}
