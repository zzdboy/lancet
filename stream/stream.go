// Copyright 2023 dudaodong@gmail.com. All rights resulterved.
// Use of this source code is governed by MIT license

// Package stream implements a sequence of elements supporting sequential and parallel aggregate operations.
// this package is an experiment to explore if stream in go can work as the way java does. it's complete, but not
// powerful like other libs
package stream

import (
	"bytes"
	"encoding/gob"

	"golang.org/x/exp/constraints"
)

// A stream should implements methods:
// type StreamI[T any] interface {

// 	// part methods of Java Stream Specification.
// 	Distinct() StreamI[T]
// 	Filter(predicate func(item T) bool) StreamI[T]
// 	FlatMap(mapper func(item T) StreamI[T]) StreamI[T]
// 	Map(mapper func(item T) T) StreamI[T]
// 	Peek(consumer func(item T)) StreamI[T]

// 	Sort(less func(a, b T) bool) StreamI[T]
// 	Max(less func(a, b T) bool) (T, bool)
// 	Min(less func(a, b T) bool) (T, bool)

// 	Limit(maxSize int) StreamI[T]
// 	Skip(n int) StreamI[T]

// 	AllMatch(predicate func(item T) bool) bool
// 	AnyMatch(predicate func(item T) bool) bool
// 	NoneMatch(predicate func(item T) bool) bool
// 	ForEach(consumer func(item T))
// 	Reduce(accumulator func(a, b T) T) T
// 	Count() int

// 	FindFirst() (T, bool)

// 	ToSlice() []T

// 	// part of methods custom extension
// 	Reverse() StreamI[T]
// 	Range(start, end int64) StreamI[T]
// 	Concat(streams ...StreamI[T]) StreamI[T]
// }

type stream[T any] struct {
	source []T
}

// Of creates a stream stream whose elements are the specified values.
func Of[T any](elems ...T) stream[T] {
	return FromSlice(elems)
}

// Generate stream where each element is generated by the provided generater function
// generater function: func() func() (item T, ok bool) {}
func Generate[T any](generator func() func() (item T, ok bool)) stream[T] {
	source := make([]T, 0)

	var zeroValue T
	for next, item, ok := generator(), zeroValue, true; ok; {
		item, ok = next()
		if ok {
			source = append(source, item)
		}
	}

	return FromSlice(source)
}

// FromSlice create stream from slice.
func FromSlice[T any](source []T) stream[T] {
	return stream[T]{source: source}
}

// FromChannel create stream from channel.
func FromChannel[T any](source <-chan T) stream[T] {
	s := make([]T, 0)

	for v := range source {
		s = append(s, v)
	}

	return FromSlice(s)
}

// FromRange create a number stream from start to end. both start and end are included. [start, end]
func FromRange[T constraints.Integer | constraints.Float](start, end, step T) stream[T] {
	if end < start {
		panic("stream.FromRange: param start should be before param end")
	} else if step <= 0 {
		panic("stream.FromRange: param step should be positive")
	}

	l := int((end-start)/step) + 1
	source := make([]T, l, l)

	for i := 0; i < l; i++ {
		source[i] = start + (T(i) * step)
	}

	return FromSlice(source)
}

// Distinct returns a stream that removes the duplicated items.
func (s stream[T]) Distinct() stream[T] {
	source := make([]T, 0)

	distinct := map[string]bool{}

	for _, v := range s.source {
		// todo: performance issue
		k := hashKey(v)
		if _, ok := distinct[k]; !ok {
			distinct[k] = true
			source = append(source, v)
		}
	}

	return FromSlice(source)
}

func hashKey(data any) string {
	buffer := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(data)
	if err != nil {
		panic("stream.hashKey: get hashkey failed")
	}
	return buffer.String()
}

// Filter returns a stream consisting of the elements of this stream that match the given predicate.
func (s stream[T]) Filter(predicate func(item T) bool) stream[T] {
	source := make([]T, 0)

	for _, v := range s.source {
		if predicate(v) {
			source = append(source, v)
		}
	}

	return FromSlice(source)
}

// Map returns a stream consisting of the elements of this stream that apply the given function to elements of stream.
func (s stream[T]) Map(mapper func(item T) T) stream[T] {
	source := make([]T, s.Count(), s.Count())

	for i, v := range s.source {
		source[i] = mapper(v)
	}

	return FromSlice(source)
}

// Peek returns a stream consisting of the elements of this stream, additionally performing the provided action on each element as elements are consumed from the resulting stream.
func (s stream[T]) Peek(consumer func(item T)) stream[T] {
	for _, v := range s.source {
		consumer(v)
	}

	return s
}

// Skip returns a stream consisting of the remaining elements of this stream after discarding the first n elements of the stream.
// If this stream contains fewer than n elements then an empty stream will be returned.
func (s stream[T]) Skip(n int) stream[T] {
	if n <= 0 {
		return s
	}

	source := make([]T, 0)
	l := len(s.source)

	if n > l {
		return FromSlice(source)
	}

	for i := n; i < l; i++ {
		source = append(source, s.source[i])
	}

	return FromSlice(source)
}

// Limit returns a stream consisting of the elements of this stream, truncated to be no longer than maxSize in length.
func (s stream[T]) Limit(maxSize int) stream[T] {
	if s.source == nil {
		return s
	}

	if maxSize < 0 {
		return FromSlice([]T{})
	}

	source := make([]T, 0, maxSize)

	for i := 0; i < len(s.source) && i < maxSize; i++ {
		source = append(source, s.source[i])
	}

	return FromSlice(source)
}

// AllMatch returns whether all elements of this stream match the provided predicate.
func (s stream[T]) AllMatch(predicate func(item T) bool) bool {
	for _, v := range s.source {
		if !predicate(v) {
			return false
		}
	}

	return true
}

// AnyMatch returns whether any elements of this stream match the provided predicate.
func (s stream[T]) AnyMatch(predicate func(item T) bool) bool {
	for _, v := range s.source {
		if predicate(v) {
			return true
		}
	}

	return false
}

// NoneMatch returns whether no elements of this stream match the provided predicate.
func (s stream[T]) NoneMatch(predicate func(item T) bool) bool {
	return !s.AnyMatch(predicate)
}

// ForEach performs an action for each element of this stream.
func (s stream[T]) ForEach(action func(item T)) {
	for _, v := range s.source {
		action(v)
	}
}

// Reduce performs a reduction on the elements of this stream, using an associative accumulation function, and returns an Optional describing the reduced value, if any.
func (s stream[T]) Reduce(init T, accumulator func(a, b T) T) T {
	for _, v := range s.source {
		init = accumulator(init, v)
	}

	return init
}

// Count returns the count of elements in the stream.
func (s stream[T]) Count() int {
	return len(s.source)
}

// ToSlice return the elements in the stream.
func (s stream[T]) ToSlice() []T {
	return s.source
}
