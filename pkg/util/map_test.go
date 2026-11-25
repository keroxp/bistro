package util_test

import (
	"slices"
	"testing"

	"github.com/loilo-inc/bistro/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestNewMap(t *testing.T) {
	m := util.NewMap(map[string]int{"first": 42, "zero": 0})
	require.Equal(t, 2, m.Size())
	require.Equal(t, 42, m.Get("first"))
	require.True(t, m.Has("zero"))

	empty := util.NewMap[string, int](nil)
	require.Zero(t, empty.Size())
	empty.Set("key", 1)
	require.Equal(t, 1, empty.Get("key"))
}

func TestMapKeys(t *testing.T) {
	var empty util.Map[string, int]
	require.Empty(t, slices.Collect(empty.Keys()))

	m := util.NewMap(map[string]int{"first": 42, "second": 99, "removed": 0})
	m.Delete("removed")
	require.ElementsMatch(t, []string{"first", "second"}, slices.Collect(m.Keys()))
	count := 0
	for key := range m.Keys() {
		require.True(t, m.Has(key))
		count++
		break
	}
	require.Equal(t, 1, count)
	m.Clear()
	require.Empty(t, slices.Collect(m.Keys()))
}

func TestMapCloneIndependence(t *testing.T) {
	for _, state := range []string{"zero value", "populated"} {
		t.Run(state, func(t *testing.T) {
			var original util.Map[string, int]
			if state == "populated" {
				original.Set("shared", 42)
			}
			clone := original.Clone()
			require.Equal(t, original.Size(), clone.Size())
			require.Equal(t, original.Get("shared"), clone.Get("shared"))
			clone.Set("shared", 99)
			clone.Set("clone only", 1)
			require.False(t, original.Has("clone only"))
			require.NotEqual(t, original.Get("shared"), clone.Get("shared"))
			original.Set("original only", 2)
			require.False(t, clone.Has("original only"))
			clone.Clear()
			require.True(t, original.Has("original only"))
		})
	}
}

func TestMapZeroValue(t *testing.T) {
	var m util.Map[string, int]
	require.Zero(t, m.Size())
	require.Zero(t, m.Get("missing"))
	value, ok := m.LookUp("missing")
	require.Zero(t, value)
	require.False(t, ok)
	require.False(t, m.Has("missing"))

	m.Delete("missing")
	m.Clear()
	require.Zero(t, m.Size())
	m.Set("key", 42)
	require.Equal(t, 42, m.Get("key"))
}

func TestMapSetAndLookUp(t *testing.T) {
	var m util.Map[string, int]
	m.Set("first", 42)
	m.Set("zero", 0)
	require.Equal(t, 2, m.Size())

	for key, want := range map[string]int{"first": 42, "zero": 0} {
		value, ok := m.LookUp(key)
		require.True(t, ok, key)
		require.Equal(t, want, value)
		require.Equal(t, want, m.Get(key))
		require.True(t, m.Has(key))
	}
	value, ok := m.LookUp("missing")
	require.Zero(t, value)
	require.False(t, ok)
	require.Zero(t, m.Get("missing"))
	require.False(t, m.Has("missing"))

	m.Set("first", 99)
	require.Equal(t, 99, m.Get("first"))
	require.Equal(t, 2, m.Size())
	require.Zero(t, m.Get("zero"))
}

func TestMapDelete(t *testing.T) {
	var m util.Map[int, string]
	m.Set(1, "first")
	m.Set(2, "second")
	m.Delete(1)
	require.Equal(t, 1, m.Size())
	require.False(t, m.Has(1))
	value, ok := m.LookUp(1)
	require.Empty(t, value)
	require.False(t, ok)
	require.Equal(t, "second", m.Get(2))

	m.Delete(1)
	m.Delete(3)
	require.Equal(t, 1, m.Size())
	m.Delete(2)
	require.Zero(t, m.Size())
	m.Set(1, "again")
	require.Equal(t, "again", m.Get(1))
}

func TestMapClear(t *testing.T) {
	var m util.Map[string, *int]
	value := 42
	m.Set("first", &value)
	m.Set("nil", nil)
	got, ok := m.LookUp("nil")
	require.Nil(t, got)
	require.True(t, ok)
	require.True(t, m.Has("nil"))

	m.Clear()
	require.Zero(t, m.Size())
	for _, key := range []string{"first", "nil"} {
		got, ok := m.LookUp(key)
		require.Nil(t, got)
		require.False(t, ok)
		require.False(t, m.Has(key))
	}
	m.Clear()
	m.Set("again", &value)
	require.Equal(t, 1, m.Size())
	require.Same(t, &value, m.Get("again"))
}

func TestMapIterEmpty(t *testing.T) {
	for _, state := range []string{"zero value", "cleared", "deleted"} {
		t.Run(state, func(t *testing.T) {
			var m util.Map[string, int]
			switch state {
			case "cleared":
				m.Set("key", 42)
				m.Clear()
			case "deleted":
				m.Set("key", 42)
				m.Delete("key")
			}
			for range m.Iter() {
				t.Fatal("empty map yielded an entry")
			}
		})
	}
}

func TestMapIterEntries(t *testing.T) {
	var m util.Map[string, int]
	m.Set("first", 42)
	m.Set("first", 99)
	m.Set("zero", 0)
	m.Set("deleted", 7)
	m.Delete("deleted")

	got := make(map[string]int)
	for key, value := range m.Iter() {
		require.NotContains(t, got, key)
		got[key] = value
	}
	require.Equal(t, map[string]int{"first": 99, "zero": 0}, got)
}

func TestMapIterEarlyStop(t *testing.T) {
	var m util.Map[string, int]
	m.Set("first", 42)
	m.Set("second", 99)

	count := 0
	for key, value := range m.Iter() {
		require.Equal(t, m.Get(key), value)
		count++
		break
	}
	require.Equal(t, 1, count)
	require.Equal(t, 2, m.Size())
}
