package promise

import (
	"errors"
	"reflect"
	"sync"
)

const (
	pending = iota
	fulfilled
	rejected
)

// A Promise is a proxy for a value not necessarily known when
// the promise is created. It allows you to associate handlers
// with an asynchronous action's eventual success value or failure reason.
// This lets asynchronous methods return values like synchronous methods:
// instead of immediately returning the final value, the asynchronous method
// returns a promise to supply the value at some point in the future.
type Promise struct {
	// A Promise is in one of these states:
	// Pending - 0. Initial state, neither fulfilled nor rejected.
	// Fulfilled - 1. Operation completed successfully.
	// Rejected - 2. Operation failed.
	state int

	// A function that is passed with the arguments resolve and reject.
	// The executor function is executed immediately by the Promise implementation,
	// passing resolve and reject functions (the executor is called
	// before the Promise constructor even returns the created object).
	// The resolve and reject functions, when called, resolve or reject
	// the promise, respectively. The executor normally initiates some
	// asynchronous work, and then, once that completes, either calls the
	// resolve function to resolve the promise or else rejects it if
	// an error or panic occurred.
	executor     func(resolve func(interface{}), reject func(error))
	iterExecutor func(wg *sync.WaitGroup, iterPromise *Promise, resolve func(interface{}), reject func(error))
	isIterable   bool

	// Appends fulfillment to the promise,
	// and returns a new promise.
	then     []func(data interface{}) interface{}
	thenDone int

	// Appends a rejection handler to the promise,
	// and returns a new promise.
	catch   []func(error error) (interface{}, error)
	finally []func()

	errorHandled bool

	// Stores the result passed to resolve()
	result interface{}

	// Stores the error passed to reject()
	error error

	// Mutex protects against data race conditions.
	mutex *sync.Mutex

	// WaitGroup allows to block until all callbacks are executed.
	wg *sync.WaitGroup

	// Iteration WaitGroup allows to block until all iterables have been processed before proceeding.
	iterWg *sync.WaitGroup
}

// New instantiates and returns a *Promise object.
func newIterPromise(iterSize int, iterExecutor func(wg *sync.WaitGroup, iterPromise *Promise, resolve func(interface{}), reject func(error))) *Promise {
	var promise = &Promise{
		state:        pending,
		iterExecutor: iterExecutor,
		then:         make([]func(interface{}) interface{}, 0),
		thenDone:     0,
		catch:        make([]func(error) (interface{}, error), 0),
		errorHandled: false,
		finally:      make([]func(), 0),
		result:       nil,
		error:        nil,
		mutex:        &sync.Mutex{},
		wg:           &sync.WaitGroup{},
		iterWg:       &sync.WaitGroup{},
		isIterable:   true,
	}

	promise.wg.Add(1)
	promise.iterWg.Add(iterSize)

	go func() {
		defer promise.handlePanic()
		promise.iterExecutor(promise.iterWg, promise, promise.resolve, promise.reject)
	}()

	return promise
}

// New instantiates and returns a *Promise object.
func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	var promise = &Promise{
		state:        pending,
		executor:     executor,
		then:         make([]func(interface{}) interface{}, 0),
		thenDone:     0,
		catch:        make([]func(error) (interface{}, error), 0),
		errorHandled: false,
		finally:      make([]func(), 0),
		result:       nil,
		error:        nil,
		mutex:        &sync.Mutex{},
		wg:           &sync.WaitGroup{},
	}

	promise.wg.Add(1)

	go func() {
		defer promise.handlePanic()
		promise.executor(promise.resolve, promise.reject)
	}()

	return promise
}

func (promise *Promise) resolve(resolution interface{}) {

	if promise.state != pending {
		return
	}

	promise.mutex.Lock()

	promise.result = resolution

	if promise.thenDone >= 0 && promise.thenDone < len(promise.then) {

		for _, value := range promise.then[promise.thenDone:] {
			r, err := promise.checkReturnedFulfilmentValue(value(promise.result))
			if err != nil {
				promise.mutex.Unlock()
				promise.reject(err)
				return
			}
			promise.thenDone++
			promise.result = r
			promise.wg.Done()
			if promise.state == rejected {
				promise.mutex.Unlock()
				return
			}
		}

	}
	defer promise.mutex.Unlock()

	for range promise.catch {
		promise.wg.Done()
	}

	promise.state = fulfilled

	promise.runFinally()

	promise.wg.Done()

}

func (promise *Promise) reject(error error) {

	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state != pending {
		return
	}

	promise.state = rejected

	expectThen := len(promise.then)

	for range promise.then {
		if expectThen-promise.thenDone <= 0 {
			break
		}
		promise.wg.Done()
		promise.thenDone++
	}

	promise.error = error

	for i, value := range promise.catch {
		v, err, _ := promise.checkReturnedRejectionValue(value(promise.error))
		if err == nil && v != nil {
			promise.error = err
			promise.errorHandled = true
			promise.result = v
			if i < len(promise.catch)-1 {
				for range promise.catch[i:] {
					promise.wg.Done()
				}
			}
			break
		} else if !promise.errorHandled {
			promise.error = err
		}
		promise.wg.Done()
	}

	promise.runFinally()

	promise.wg.Done()

}

func (promise *Promise) handlePanic() {
	var r = recover()
	if r != nil {
		promise.reject(errors.New(r.(string)))
	}
}

func (promise *Promise) runFinally() {

	for _, value := range promise.finally {
		value()
		promise.wg.Done()
	}
}

// Then appends fulfillment handler to the promise, and returns a new promise.
func (promise *Promise) Then(fulfillment func(data interface{}) interface{}) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state == pending {
		promise.wg.Add(1)
		promise.then = append(promise.then, fulfillment)
	} else if promise.state == fulfilled {
		promise.result = fulfillment(promise.result)
	}

	return promise
}

// Catch appends a rejection handler callback to the promise, and returns a new promise.
func (promise *Promise) Catch(rejection func(error error) (interface{}, error)) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state == pending {
		promise.wg.Add(1)
		promise.catch = append(promise.catch, rejection)
	} else if promise.state == rejected {
		if promise.errorHandled {
			return promise
		}
		v, err, _ := promise.checkReturnedRejectionValue(rejection(promise.error))
		if err == nil && v != nil {
			promise.error = err
			promise.errorHandled = true
			promise.result = v
			return promise
		} else if !promise.errorHandled {
			promise.error = err
		}
	}

	return promise
}

// Spread is like then but if the result is a slice the values are spread out into the thenable function...
func (promise *Promise) Spread(spreadFn func(vals ...interface{}) interface{}) *Promise {
	return promise.Then(func(data interface{}) interface{} {
		if reflect.TypeOf(data).Kind() != reflect.Slice {
			return spreadFn(data)
		} else {
			odata := data.([]interface{})
			return spreadFn(odata...)
		}
	})
}

// Each will take an iterable result from a thenable and iterate the results and run the eachFn on the item
func (promise *Promise) Each(eachFn PromiseIteratorFunc) *Promise {
	return promise.Then(func(data interface{}) interface{} {
		if reflect.TypeOf(data).Kind() != reflect.Slice {
			_, err := iteratorEstablishFunc(data, eachFn)
			if err != nil {
				promise.reject(err)
				return nil
			}
			return nil
		} else {
			odata := data.([]interface{})
			p := Each(odata, eachFn)
			_, err := p.Yield()
			if err != nil {
				promise.reject(err)
				return nil
			}
			return nil
		}
	})
}

// Map will take an iterable result from a thenable and Map it to an array in parallel (like each but actually returns stuff)
func (promise *Promise) Map(mapFn PromiseIteratorFunc) *Promise {
	return promise.Then(func(data interface{}) interface{} {
		if reflect.TypeOf(data).Kind() != reflect.Slice {
			v, err := iteratorEstablishFunc(data, mapFn)
			if err != nil {
				promise.reject(err)
				return nil
			}
			return v
		} else {
			odata := data.([]interface{})
			p := Map(odata, mapFn)
			v, err := p.Yield()
			if err != nil {
				promise.reject(err)
				return nil
			}
			return v
		}
	})
}

// MapSeries will take an iterable result from a thenable and Map it to an array in series (like each but actually returns stuff)
func (promise *Promise) MapSeries(mapFn PromiseIteratorFunc) *Promise {
	return promise.Then(func(data interface{}) interface{} {
		if reflect.TypeOf(data).Kind() != reflect.Slice {
			v, err := iteratorEstablishFunc(data, mapFn)
			if err != nil {
				promise.reject(err)
				return nil
			}
			return v
		} else {
			odata := data.([]interface{})
			p := MapSeries(odata, mapFn)
			v, err := p.Yield()
			if err != nil {
				promise.reject(err)
				return nil
			}
			return v
		}
	})
}

// Catch appends a rejection handler callback to the promise, and returns a new promise.
func (promise *Promise) Finally(finallyFunc func()) *Promise {
	promise.mutex.Lock()
	defer promise.mutex.Unlock()

	if promise.state == pending {
		promise.wg.Add(1)
		promise.finally = append(promise.finally, finallyFunc)
	} else if promise.state != pending {
		finallyFunc()
	}

	return promise
}

// Yield will await for a resolution or rejection via Await and Yield the result...
func (promise *Promise) Yield() (interface{}, error) {
	promise.Await()
	//get the values and return
	return promise.result, promise.error
}

// Await is a blocking function that waits for all callbacks to be executed.
func (promise *Promise) Await() {
	if promise.isIterable {
		promise.iterWg.Wait()
	}
	promise.wg.Wait()
}

// AwaitAll is a blocking function that waits for a number of promises to resolve
func AwaitAll(promises ...*Promise) {
	for _, promise := range promises {
		promise.Await()
	}
}
