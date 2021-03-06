package nds_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/qedus/nds"

	"errors"

	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"appengine/memcache"
)

func TestGetMultiStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiStructPtr(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiInterface(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	response = make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}
}

func TestGetMultiPropertyLoadSaver(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int
	}

	keys := []*datastore.Key{}
	entities := []datastore.PropertyList{}

	for i := 1; i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", int64(i), nil))

		pl := datastore.PropertyList{}
		if err := nds.SaveStruct(&testEntity{i}, &pl); err != nil {
			t.Fatal(err)
		}
		entities = append(entities, pl)
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Prime the cache.
	uncachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, uncachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, uncachedEntities[i]) {
			t.Fatal("uncachedEntities not equal", e, uncachedEntities[i])
		}
	}

	// Use cache.
	cachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, cachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, cachedEntities[i]) {
			t.Fatal("cachedEntities not equal", e, cachedEntities[i])
		}
	}

	// We know the datastore supports property load saver but we need to make
	// sure that memcache does by ensuring memcache does not error when we
	// change to fetching with structs.
	// Do this by making sure the datastore is not called on this following
	// GetMulti as memcache should have worked.
	nds.SetDatastoreGetMulti(func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		if len(keys) != 0 {
			return errors.New("should not be called")
		}
		return nil
	})
	defer func() {
		nds.SetDatastoreGetMulti(datastore.GetMulti)
	}()
	tes := make([]testEntity, len(entities))
	if err := nds.GetMulti(c, keys, tes); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiNoKeys(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}

	if err := nds.GetMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiInterfaceError(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	// No errors expected.
	response := []interface{}{&testEntity{}, &testEntity{}}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	// Errors expected.
	response = []interface{}{&testEntity{}, testEntity{}}
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected invalid entity type error")
	}
}

// This is just used to ensure interfaces don't currently work.
type readerTestEntity struct {
	IntVal int
}

func (rte readerTestEntity) Read(p []byte) (n int, err error) {
	return 1, nil
}

var _ io.Reader = readerTestEntity{}

func newReaderTestEntity() io.Reader {
	return readerTestEntity{}
}

func TestGetArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	if err := nds.Get(c, nil, &testEntity{}); err == nil {
		t.Fatal("expected error for nil key")
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	if err := nds.Get(c, key, nil); err == nil {
		t.Fatal("expected error for nil value")
	}

	if err := nds.Get(c, key, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for datastore.PropertyList")
	}

	if err := nds.Get(c, key, testEntity{}); err == nil {
		t.Fatal("expected error for struct")
	}

	rte := newReaderTestEntity()
	if err := nds.Get(c, key, rte); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetMultiArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	keys := []*datastore.Key{key}
	val := testEntity{}
	if err := nds.GetMulti(c, keys, nil); err == nil {
		t.Fatal("expected error for nil vals")
	}
	structVals := []testEntity{val}
	if err := nds.GetMulti(c, nil, structVals); err == nil {
		t.Fatal("expected error for nil keys")
	}

	if err := nds.GetMulti(c, keys, []testEntity{}); err == nil {
		t.Fatal("expected error for unequal keys and vals")
	}

	if err := nds.GetMulti(c, keys, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for propertyList")
	}

	rte := newReaderTestEntity()
	if err := nds.GetMulti(c, keys, []io.Reader{rte}); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetSliceProperty(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVals []int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	intVals := []int64{0, 1, 2, 3}
	val := &testEntity{intVals}

	if _, err := nds.Put(c, key, val); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	newVal := &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}

	// Get from memcache.
	newVal = &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}
}

func TestGetMultiNoPropertyList(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	pl := datastore.PropertyList{datastore.Property{}}

	if err := nds.GetMulti(c, keys, pl); err == nil {
		t.Fatal("expecting no PropertyList error")
	}
}

func TestGetMultiNonStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	vals := []int{12}

	if err := nds.GetMulti(c, keys, vals); err == nil {
		t.Fatal("expecting unsupported vals type")
	}
}

func TestGetMultiLockReturnEntitySetValueFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Fail to unmarshal test.
	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		pl := datastore.PropertyList{
			datastore.Property{"One", 1, false, false},
		}
		value, err := nds.MarshalPropertyList(pl)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.EntityItem
		items[keys[0]].Value = value
		items[keys[1]].Flags = nds.EntityItem
		items[keys[1]].Value = value
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnEntity(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		pl := datastore.PropertyList{
			datastore.Property{"IntVal", int64(5), false, false},
		}
		value, err := nds.MarshalPropertyList(pl)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.EntityItem
		items[keys[0]].Value = value
		items[keys[1]].Flags = nds.EntityItem
		items[keys[1]].Value = value
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if 5 != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnUnknown(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}

		// Unknown lock values.
		items[keys[0]].Flags = 23
		items[keys[1]].Flags = 24
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnMiss(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}

		// Remove one item between memcache Add and Get.
		delete(items, keys[0])
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiPaths(t *testing.T) {
	expectedErr := errors.New("expected error")

	type memcacheGetMultiFunc func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error)
	memcacheGetMultiFail := func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		return nil, expectedErr
	}

	type memcacheAddMultiFunc func(c appengine.Context,
		items []*memcache.Item) error
	memcacheAddMultiFail := func(c appengine.Context,
		items []*memcache.Item) error {
		return expectedErr
	}

	type memcacheCompareAndSwapMultiFunc func(c appengine.Context,
		items []*memcache.Item) error
	memcacheCompareAndSwapMultiFail := func(c appengine.Context,
		items []*memcache.Item) error {
		return expectedErr
	}

	type datastoreGetMultiFunc func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error
	datastoreGetMultiFail := func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		return expectedErr
	}

	type marshalFunc func(pl datastore.PropertyList) ([]byte, error)
	marshalFail := func(pl datastore.PropertyList) ([]byte, error) {
		return nil, expectedErr
	}

	type unmarshalFunc func(data []byte, pl *datastore.PropertyList) error
	/*
	   unmarshalFail := func(data []byte, pl *datastore.PropertyList) error {
	       return expectedErr
	   }
	*/

	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keysVals := func(c appengine.Context, count int64) (
		[]*datastore.Key, []testEntity) {

		keys, vals := make([]*datastore.Key, count), make([]testEntity, count)
		for i := int64(0); i < count; i++ {
			keys[i] = datastore.NewKey(c, "Entity", "", i+1, nil)
			vals[i] = testEntity{i + 1}
		}
		return keys, vals
	}

	tests := []struct {
		description string

		// Number of keys used to as GetMulti params.
		keyCount int64

		// Number of times GetMulti is called.
		callCount int

		// There are 2 memcacheGetMulti calls for every GetMulti call.
		memcacheGetMultis           []memcacheGetMultiFunc
		memcacheAddMulti            memcacheAddMultiFunc
		memcacheCompareAndSwapMulti memcacheCompareAndSwapMultiFunc

		datastoreGetMulti datastoreGetMultiFunc

		marshal marshalFunc
		// There are 2 unmarshal calls for every GetMultiCall.
		unmarshals []unmarshalFunc

		expectedErrs []error
	}{
		{
			"no errors",
			20,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
		{
			"datastore unknown error",
			2,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastoreGetMultiFail,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{expectedErr},
		},
		{
			"datastore unknown multierror",
			2,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			func(c appengine.Context,
				keys []*datastore.Key, vals interface{}) error {

				me := make(appengine.MultiError, len(keys))
				for i := range me {
					me[i] = expectedErr
				}
				return me
			},
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{
				appengine.MultiError{expectedErr, expectedErr},
			},
		},
		{
			"marshal error",
			5,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			marshalFail,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
		{
			"total memcache fail",
			20,
			1,
			[]memcacheGetMultiFunc{
				memcacheGetMultiFail,
				memcacheGetMultiFail,
			},
			memcacheAddMultiFail,
			memcacheCompareAndSwapMultiFail,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
		{
			"lock memcache fail",
			20,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				memcacheGetMultiFail,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
		{
			"memcache corrupt",
			2,
			2,
			[]memcacheGetMultiFunc{
				// Charge memcache.
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
				// Corrupt memcache.
				func(c appengine.Context, keys []string) (
					map[string]*memcache.Item, error) {
					items, err := memcache.GetMulti(c, keys)
					// Corrupt items.
					for _, item := range items {
						item.Value = []byte("corrupt string")
					}
					return items, err
				},
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil, nil},
		},
		{
			"memcache flag corrupt",
			2,
			2,
			[]memcacheGetMultiFunc{
				// Charge memcache.
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
				// Corrupt memcache flags.
				func(c appengine.Context, keys []string) (
					map[string]*memcache.Item, error) {
					items, err := memcache.GetMulti(c, keys)
					// Corrupt flags with unknown number.
					for _, item := range items {
						item.Flags = 56
					}
					return items, err
				},
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil, nil},
		},
		{
			"lock memcache value fail",
			20,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				func(c appengine.Context, keys []string) (
					map[string]*memcache.Item, error) {
					items, err := memcache.GetMulti(c, keys)
					// Corrupt flags with unknown number.
					for _, item := range items {
						item.Value = []byte("corrupt value")
					}
					return items, err
				},
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
		{
			"lock memcache value none item",
			2,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				func(c appengine.Context, keys []string) (
					map[string]*memcache.Item, error) {
					items, err := memcache.GetMulti(c, keys)
					// Corrupt flags with unknown number.
					for _, item := range items {
						item.Flags = nds.NoneItem
					}
					return items, err
				},
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{
				appengine.MultiError{
					datastore.ErrNoSuchEntity,
					datastore.ErrNoSuchEntity,
				},
			},
		},
		{
			"memcache get no entity unmarshal fail",
			2,
			1,
			[]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				func(c appengine.Context, keys []string) (
					map[string]*memcache.Item, error) {
					items, err := memcache.GetMulti(c, keys)
					// Corrupt flags with unknown number.
					for _, item := range items {
						item.Flags = nds.EntityItem
					}
					return items, err
				},
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,
			nds.MarshalPropertyList,
			[]unmarshalFunc{
				nds.UnmarshalPropertyList,
				nds.UnmarshalPropertyList,
			},
			[]error{nil},
		},
	}

	for _, test := range tests {
		t.Log("Start", test.description)

		keys, putVals := keysVals(c, test.keyCount)
		if _, err := nds.PutMulti(c, keys, putVals); err != nil {
			t.Fatal(err)
		}

		memcacheGetChan := make(chan memcacheGetMultiFunc,
			len(test.memcacheGetMultis))

		for _, fn := range test.memcacheGetMultis {
			memcacheGetChan <- fn
		}

		nds.SetMemcacheGetMulti(func(c appengine.Context, keys []string) (
			map[string]*memcache.Item, error) {
			fn := <-memcacheGetChan
			return fn(c, keys)
		})

		nds.SetMemcacheAddMulti(test.memcacheAddMulti)
		nds.SetMemcacheCompareAndSwapMulti(test.memcacheCompareAndSwapMulti)

		nds.SetDatastoreGetMulti(test.datastoreGetMulti)

		nds.SetMarshal(test.marshal)

		unmarshalChan := make(chan unmarshalFunc,
			len(test.unmarshals))

		for _, fn := range test.unmarshals {
			unmarshalChan <- fn
		}

		nds.SetUnmarshal(func(data []byte, pl *datastore.PropertyList) error {
			fn := <-unmarshalChan
			return fn(data, pl)
		})

		for i := 0; i < test.callCount; i++ {
			getVals := make([]testEntity, test.keyCount)
			err := nds.GetMulti(c, keys, getVals)

			expectedErr := test.expectedErrs[i]

			if expectedErr == nil {
				if err != nil {
					t.Fatal(err)
				}

				for i := range getVals {
					if getVals[i].IntVal != putVals[i].IntVal {
						t.Fatal("incorrect IntVal")
					}
				}
				continue
			}

			if err == nil {
				t.Fatal("expected error")
			}
			expectedMultiErr, isMultiErr := expectedErr.(appengine.MultiError)

			if isMultiErr {
				me, ok := err.(appengine.MultiError)
				if !ok {
					t.Fatal("expected appengine.MultiError but got", err)
				}

				if len(me) != len(expectedMultiErr) {
					t.Fatal("appengine.MultiError length incorrect")
				}

				for i, e := range me {
					if e != expectedMultiErr[i] {
						t.Fatal("non matching errors", e, expectedMultiErr[i])
					}

					if e == nil {
						if getVals[i].IntVal != putVals[i].IntVal {
							t.Fatal("incorrect IntVal")
						}
					}
				}
			}
		}

		// Reset App Engine API calls.
		nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)
		nds.SetMemcacheAddMulti(nds.ZeroMemcacheAddMulti)
		nds.SetMemcacheCompareAndSwapMulti(nds.ZeroMemcacheCompareAndSwapMulti)
		nds.SetDatastoreGetMulti(datastore.GetMulti)
		nds.SetMarshal(nds.MarshalPropertyList)
		nds.SetUnmarshal(nds.UnmarshalPropertyList)

		if err := nds.DeleteMulti(c, keys); err != nil {
			t.Fatal(err)
		}
		t.Log("End", test.description)
	}
}
