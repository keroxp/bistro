package util

import (
	"iter"
	"slices"
)

type OrderedSet[T comparable] struct {
	set        Map[T, struct{}]
	values     []T
	needsOrder bool
}

func NewOrderedSet[T comparable](values ...T) OrderedSet[T] {
	os := OrderedSet[T]{}
	for _, v := range values {
		os.Add(v)
	}
	return os
}

func (os *OrderedSet[T]) Add(value ...T) {
	for _, v := range value {
		if !os.set.Has(v) {
			os.set.Set(v, struct{}{})
			os.needsOrder = true
		}
	}
}

func (os *OrderedSet[T]) Remove(value T) {
	if os.set.Has(value) {
		os.set.Delete(value)
		os.needsOrder = true
	}
}

func (os *OrderedSet[T]) Has(value T) bool {
	return os.set.Has(value)
}

func (os *OrderedSet[T]) Size() int {
	return os.set.Size()
}

func (os *OrderedSet[T]) Clear() {
	os.set.Clear()
	os.needsOrder = true
}

func (os *OrderedSet[T]) At(index int) T {
	if os.needsOrder {
		os.reorder()
	}
	return os.values[index]
}

func (os *OrderedSet[T]) Clone() OrderedSet[T] {
	return OrderedSet[T]{
		set:        os.set.Clone(),
		needsOrder: true,
	}
}

func (os *OrderedSet[T]) reorder() {
	os.needsOrder = false
	os.values = make([]T, 0, os.set.Size())
	for k := range os.set.Keys() {
		os.values = append(os.values, k)
	}
}

func (os *OrderedSet[T]) Values() iter.Seq[T] {
	if os.needsOrder {
		os.reorder()
	}
	return func(yield func(T) bool) {
		for _, v := range os.values {
			if !yield(v) {
				return
			}
		}
	}
}

func (os *OrderedSet[T]) Slice() []T {
	return slices.Collect(os.Values())
}
