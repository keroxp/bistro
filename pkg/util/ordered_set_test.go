package util_test

import (
	"slices"
	"testing"

	"github.com/loilo-inc/bistro/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestOrderedSetZeroValue(t *testing.T) {
	var set util.OrderedSet[string]
	require.Zero(t, set.Size())
	require.False(t, set.Has("missing"))
	require.Empty(t, set.Slice())
	set.Remove("missing")
	set.Clear()
	require.Empty(t, slices.Collect(set.Values()))
	set.Add("first")
	require.Equal(t, "first", set.At(0))
	require.Equal(t, "first", set.At(0))
}

func TestOrderedSetMutations(t *testing.T) {
	set := util.NewOrderedSet("first", "second", "first")
	require.Equal(t, 2, set.Size())
	require.ElementsMatch(t, []string{"first", "second"}, set.Slice())
	set.Add("second", "third", "third")
	require.Equal(t, 3, set.Size())
	require.True(t, set.Has("third"))
	require.ElementsMatch(t, []string{"first", "second", "third"}, set.Slice())
	set.Remove("second")
	set.Remove("second")
	set.Remove("missing")
	require.False(t, set.Has("second"))
	require.Equal(t, 2, set.Size())
	require.ElementsMatch(t, []string{"first", "third"}, set.Slice())
	set.Add("second")
	require.ElementsMatch(t, []string{"first", "second", "third"}, set.Slice())
	set.Clear()
	require.Zero(t, set.Size())
	require.False(t, set.Has("first"))
	require.Empty(t, set.Slice())
	set.Add("reused")
	require.Equal(t, []string{"reused"}, set.Slice())
}

func TestOrderedSetIndexedIteration(t *testing.T) {
	set := util.NewOrderedSet("first", "second", "third")
	first := set.At(0)
	values := set.Slice()
	require.Equal(t, first, values[0])
	require.Equal(t, values, slices.Collect(set.Values()))
	for i, value := range values {
		require.Equal(t, value, set.At(i))
	}
	count := 0
	for value := range set.Values() {
		require.Equal(t, first, value)
		count++
		break
	}
	require.Equal(t, 1, count)
	require.Equal(t, values, set.Slice())

	values[0] = "changed"
	require.Equal(t, first, set.At(0), "Slice must not expose the set's cached values")
}

func TestOrderedSetCloneIndependence(t *testing.T) {
	for _, state := range []string{"zero value", "before iteration", "after iteration"} {
		t.Run(state, func(t *testing.T) {
			var original util.OrderedSet[string]
			if state != "zero value" {
				original.Add("first", "second")
			}
			if state == "after iteration" {
				require.Len(t, original.Slice(), 2)
			}
			clone := original.Clone()
			require.ElementsMatch(t, original.Slice(), clone.Slice())
			clone.Remove("first")
			clone.Add("clone only")
			original.Add("original only")
			require.False(t, original.Has("clone only"))
			require.False(t, clone.Has("original only"))
			if state != "zero value" {
				require.True(t, original.Has("first"))
			}
			original.Clear()
			require.True(t, clone.Has("clone only"))
			clone.Clear()
			require.Empty(t, clone.Slice())
		})
	}
}
