package dstest

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	dstore "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

func SubtestBasicPutGet(t *testing.T, ds dstore.Datastore) {
	k := dstore.NewKey("foo")
	val := []byte("Hello Datastore!")

	err := ds.Put(k, val)
	if err != nil {
		t.Fatal("error putting to datastore: ", err)
	}

	have, err := ds.Has(k)
	if err != nil {
		t.Fatal("error calling has on key we just put: ", err)
	}

	if !have {
		t.Fatal("should have key foo, has returned false")
	}

	size, err := ds.GetSize(k)
	if err != nil {
		t.Fatal("error getting size after put: ", err)
	}
	if size != len(val) {
		t.Fatalf("incorrect size: expected %d, got %d", len(val), size)
	}

	out, err := ds.Get(k)
	if err != nil {
		t.Fatal("error getting value after put: ", err)
	}

	if !bytes.Equal(out, val) {
		t.Fatal("value received on get wasnt what we expected:", out)
	}

	have, err = ds.Has(k)
	if err != nil {
		t.Fatal("error calling has after get: ", err)
	}

	if !have {
		t.Fatal("should have key foo, has returned false")
	}

	size, err = ds.GetSize(k)
	if err != nil {
		t.Fatal("error getting size after get: ", err)
	}
	if size != len(val) {
		t.Fatalf("incorrect size: expected %d, got %d", len(val), size)
	}

	err = ds.Delete(k)
	if err != nil {
		t.Fatal("error calling delete: ", err)
	}

	have, err = ds.Has(k)
	if err != nil {
		t.Fatal("error calling has after delete: ", err)
	}

	if have {
		t.Fatal("should not have key foo, has returned true")
	}

	size, err = ds.GetSize(k)
	switch err {
	case dstore.ErrNotFound:
	case nil:
		t.Fatal("expected error getting size after delete")
	default:
		t.Fatal("wrong error getting size after delete: ", err)
	}
	if size != -1 {
		t.Fatal("expected missing size to be -1")
	}
}

func SubtestNotFounds(t *testing.T, ds dstore.Datastore) {
	badk := dstore.NewKey("notreal")

	val, err := ds.Get(badk)
	if err != dstore.ErrNotFound {
		t.Fatal("expected ErrNotFound for key that doesnt exist, got: ", err)
	}

	if val != nil {
		t.Fatal("get should always return nil for not found values")
	}

	have, err := ds.Has(badk)
	if err != nil {
		t.Fatal("error calling has on not found key: ", err)
	}
	if have {
		t.Fatal("has returned true for key we don't have")
	}

	size, err := ds.GetSize(badk)
	switch err {
	case dstore.ErrNotFound:
	case nil:
		t.Fatal("expected error getting size after delete")
	default:
		t.Fatal("wrong error getting size after delete: ", err)
	}
	if size != -1 {
		t.Fatal("expected missing size to be -1")
	}
}

func SubtestOrder(t *testing.T, ds dstore.Datastore) {
	test := func(orders ...dsq.Order) {
		var types []string
		for _, o := range orders {
			types = append(types, fmt.Sprintf("%T", o))
		}
		name := strings.Join(types, ">")
		t.Run(name, func(t *testing.T) {
			subtestQuery(t, ds, dsq.Query{
				Orders: orders,
			})
		})
	}
	test(dsq.OrderByKey{})
	test(new(dsq.OrderByKey))
	test(dsq.OrderByKeyDescending{})
	test(new(dsq.OrderByKeyDescending))
	test(dsq.OrderByValue{})
	test(dsq.OrderByValue{}, dsq.OrderByKey{})
	test(dsq.OrderByFunction(func(a, b dsq.Entry) int {
		return bytes.Compare(a.Value, b.Value)
	}))
}

func SubtestManyKeysAndQuery(t *testing.T, ds dstore.Datastore) {
	subtestQuery(t, ds, dsq.Query{KeysOnly: true})
}

// need a custom test filter to test the "fallback" filter case for unknown
// filters.
type testFilter struct{}

func (testFilter) Filter(e dsq.Entry) bool {
	return len(e.Key)%2 == 0
}

func SubtestFilter(t *testing.T, ds dstore.Datastore) {
	test := func(filters ...dsq.Filter) {
		var types []string
		for _, o := range filters {
			types = append(types, fmt.Sprintf("%T", o))
		}
		name := strings.Join(types, ">")
		t.Run(name, func(t *testing.T) {
			subtestQuery(t, ds, dsq.Query{
				Filters: filters,
			})
		})
	}
	test(dsq.FilterKeyCompare{
		Op:  dsq.Equal,
		Key: "/0key0",
	})

	test(dsq.FilterKeyCompare{
		Op:  dsq.LessThan,
		Key: "/2",
	})

	test(&dsq.FilterKeyCompare{
		Op:  dsq.Equal,
		Key: "/0key0",
	})

	test(dsq.FilterKeyPrefix{
		Prefix: "/0key0",
	})

	test(&dsq.FilterKeyPrefix{
		Prefix: "/0key0",
	})

	test(dsq.FilterValueCompare{
		Op:    dsq.LessThan,
		Value: randValue(),
	})

	test(new(testFilter))
}

func randValue() []byte {
	value := make([]byte, 64)
	rand.Read(value)
	return value
}

func subtestQuery(t *testing.T, ds dstore.Datastore, q dsq.Query) {
	var input []dsq.Entry
	count := 100
	for i := 0; i < count; i++ {
		s := fmt.Sprintf("%dkey%d", i, i)
		key := dstore.NewKey(s).String()
		value := randValue()
		input = append(input, dsq.Entry{
			Key:   key,
			Value: value,
		})
	}

	t.Logf("putting %d values", count)
	for i, e := range input {
		err := ds.Put(dstore.RawKey(e.Key), e.Value)
		if err != nil {
			t.Fatalf("error on put[%d]: %s", i, err)
		}
	}

	t.Log("getting values back")
	for i, e := range input {
		val, err := ds.Get(dstore.RawKey(e.Key))
		if err != nil {
			t.Fatalf("error on get[%d]: %s", i, err)
		}

		if !bytes.Equal(val, e.Value) {
			t.Fatal("input value didnt match the one returned from Get")
		}
	}

	t.Log("querying values")
	resp, err := ds.Query(q)
	if err != nil {
		t.Fatal("calling query: ", err)
	}

	t.Log("aggregating query results")
	actual, err := resp.Rest()
	if err != nil {
		t.Fatal("query result error: ", err)
	}

	t.Log("verifying query output")
	expected, err := dsq.NaiveQueryApply(q, dsq.ResultsWithEntries(q, input)).Rest()
	if err != nil {
		t.Fatal("naive query error: ", err)
	}
	if len(actual) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(actual))
	}
	if len(q.Orders) == 0 {
		dsq.Sort([]dsq.Order{dsq.OrderByKey{}}, actual)
		dsq.Sort([]dsq.Order{dsq.OrderByKey{}}, expected)
	}
	for i := range actual {
		if actual[i].Key != expected[i].Key {
			t.Errorf("for result %d, expected key %q, got %q", i, expected[i].Key, actual[i].Key)
			continue
		}
		if !q.KeysOnly && !bytes.Equal(actual[i].Value, expected[i].Value) {
			t.Errorf("value mismatch for result %d (key=%q)", i, expected[i].Key)
		}

	}

	t.Log("deleting all keys")
	for _, e := range input {
		if err := ds.Delete(dstore.RawKey(e.Key)); err != nil {
			t.Fatal(err)
		}
	}
}
