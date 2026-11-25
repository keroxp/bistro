package util

import (
	"iter"
	"slices"
)

type OrderedSet[T comparable] struct {
	set    Map[T, int]
	values []T
}

func NewOrderedSet[T comparable](values ...T) OrderedSet[T] {
	os := OrderedSet[T]{}
	for _, v := range values {
		os.Add(v)
	}
	return os
}

func (o *OrderedSet[T]) Add(value ...T) {
	for _, v := range value {
		if o.set.Has(v) {
			continue
		}
		idx := o.set.Size()
		o.set.Set(v, idx)
		o.values = append(o.values, v)
	}
}

func (o *OrderedSet[T]) Remove(value T) {
	if o.set.Has(value) {
		idx := o.set.Get(value)
		o.set.Delete(value)
		o.values = append(o.values[:idx], o.values[idx+1:]...)
		for i := idx; i < len(o.values); i++ {
			o.set.Set(o.values[i], i)
		}
	}
}

func (o *OrderedSet[T]) Has(value T) bool {
	return o.set.Has(value)
}

func (o *OrderedSet[T]) Size() int {
	return o.set.Size()
}

func (o *OrderedSet[T]) Clear() {
	o.set.Clear()
	o.values = nil
}

func (o *OrderedSet[T]) At(index int) T {
	return o.values[index]
}

func (o *OrderedSet[T]) Clone() OrderedSet[T] {
	return OrderedSet[T]{
		set:    o.set.Clone(),
		values: slices.Clone(o.values),
	}
}

func (o *OrderedSet[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range o.values {
			if !yield(v) {
				return
			}
		}
	}
}

func (o *OrderedSet[T]) Slice() []T {
	return slices.Clone(o.values)
}
