package promise

import (
	"errors"
	"log"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve(nil)
	})

	if promise == nil {
		t.Fatal("PROMISE IS NIL")
	}
}

func TestPromise_Then(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		resolve(1 + 1)
	})

	promise.
		Then(func(data interface{}) interface{} {
			return data.(int) + 1
		}).
		Then(func(data interface{}) interface{} {
			log.Println(data)
			if data.(int) != 3 {
				t.Fatal("RESULT DOES NOT PROPAGATE")
			}
			return nil
		})

	promise.Catch(func(err error) (interface{}, error) {
		t.Fatal("CATCH TRIGGERED IN .THEN TEST")
		return nil, nil
	})

	promise.Await()
}

func TestPromise_Catch(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		reject(errors.New("very serious error"))
	})

	promise.
		Catch(func(err error) (interface{}, error) {
			if err.Error() == "very serious error" {
				return nil, errors.New("dealing with error at this stage")
			}
			return nil, nil
		}).
		Catch(func(err error) (interface{}, error) {
			if err.Error() != "dealing with error at this stage" {
				t.Fatal("ERROR DOES NOT PROPAGATE")
			} else {
				log.Println(err.Error())
			}
			return nil, nil
		})

	promise.Then(func(data interface{}) interface{} {
		t.Fatal("THEN TRIGGERED IN .CATCH TEST")
		return nil
	})

	promise.Await()
}

func TestPromise_Panic(t *testing.T) {
	var promise = New(func(resolve func(interface{}), reject func(error)) {
		panic("much panic")
	})

	promise.
		Then(func(data interface{}) interface{} {
			t.Fatal("THEN TRIGGERED")
			return nil
		}).
		Catch(func(err error) (interface{}, error) {
			log.Println("Panic Recovered:", err.Error())
			return nil, nil
		})

	promise.Await()
}

func TestPromise_Await(t *testing.T) {
	var promises = make([]*Promise, 10)

	for x := 0; x < 10; x++ {
		var promise = New(func(resolve func(interface{}), reject func(error)) {
			resolve(time.Now())
		})

		promise.Then(func(data interface{}) interface{} {
			return data.(time.Time).Add(time.Second).Nanosecond()
		})

		promises[x] = promise
		log.Println("Added", x+1)
	}

	log.Println("Waiting")

	AwaitAll(promises...)

	for _, promise := range promises {
		promise.Then(func(data interface{}) interface{} {
			log.Println(data)
			return nil
		})
	}
}

func TestPromise_Map(t *testing.T) {
	var items = []interface{}{"hello", "world"}

	promise := Map(items, func(item interface{}) (interface{}, error) {
		//log.Println(item)
		return item.(string) + "_FROM_B", nil
	})

	v, err := promise.Yield()
	println(v, err)
}

func TestPromise_MapWithPromises(t *testing.T) {
	var items = []interface{}{"hello", New(func(resolve func(interface{}), reject func(error)) {
		resolve("Boom")
	})}

	promise := Map(items, func(item interface{}) (interface{}, error) {
		//log.Println(item)
		return item.(string) + "_FROM_B", nil
	})

	v, err := promise.Yield()
	println(v, err)
}

func TestPromise_Spread(t *testing.T) {
	var items = []interface{}{"hello", New(func(resolve func(interface{}), reject func(error)) {
		resolve("Boom")
	})}

	promise := Map(items, func(item interface{}) (interface{}, error) {
		//log.Println(item)
		return item.(string) + "_FROM_B", nil
	})

	promise.Spread(func(vals ...interface{}) interface{} {
		return vals[0].(string) + " -> " + vals[1].(string)
	})

	v, err := promise.Yield()
	println(v.(string), err)
}

func TestPromise_ChildPromises(t *testing.T) {
	promise := New(func(resolve func(interface{}), reject func(error)) {

		resolve("Outer Promise Resolved")

	}).Then(func(data interface{}) interface{} {
		return New(func(resolve func(interface{}), reject func(error)) {
			resolve("Inner Promise Resolved:" + data.(string))
		})
	})

	v, err := promise.Yield()
	println(v.(string), err)
}

func TestPromise_ChildPromiseFailed(t *testing.T) {
	promise := New(func(resolve func(interface{}), reject func(error)) {

		resolve("Outer Promise Resolved")

	}).Then(func(data interface{}) interface{} {
		println("Got Resolved", data.(string))
		return New(func(resolve func(interface{}), reject func(error)) {
			reject(errors.New("Inner Promise Rejected:" + data.(string)))
		})
	})

	v, err := promise.Yield()
	println(v.(string), err.Error())
}

func TestPromise_Prop(t *testing.T) {
	var items = map[string]interface{}{
		"aa": "boom1",
		"bb": New(func(resolve func(interface{}), reject func(error)) {
			resolve("boom2")
		}),
	}

	promise := Props(items, func(item interface{}) (interface{}, error) {
		//log.Println(item)
		return item.(string) + "_FROM_B", nil
	})

	v, err := promise.Yield()
	log.Println(v, err)
}
