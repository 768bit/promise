package promise

import (
	"context"
	"sync"
)

func Each(items []interface{}, iteratorFunc PromiseIteratorFunc) *Promise {

	p := New(func(resolve func(interface{}), reject func(error)) {

		for _, item := range items {
			_, err := iteratorEstablishFunc(item, iteratorFunc)
			if err != nil {
				reject(err)
				return
			}
		}

		resolve(nil)

	})
	return p

}

func Map(items []interface{}, iteratorFunc PromiseIteratorFunc) *Promise {
	//in parallel run the iteratorFunc against the items and return a same-order array in the resolve function or error the wrapped promise

	p := newIterPromise(len(items), func(wg *sync.WaitGroup, iterProm *Promise, resolve func(interface{}), reject func(error)) {

		ctx, cancel := context.WithCancel(context.Background())
		//create the destination item...
		mutex := &sync.Mutex{}
		ret := make([]interface{}, len(items))
		defer cancel()

		for i, item := range items {
			go func(inItem interface{}, index int) {
				vchan, echan := iteratorEstablishFuncChan(inItem, iteratorFunc)
				defer close(vchan)
				defer close(echan)
				select {
				case err := <-echan:
					if err != nil {
						cancel()
						iterProm.state = rejected
						go reject(err)
						wg.Done()
						return
					}
				case v := <-vchan:
					mutex.Lock()
					defer mutex.Unlock()
					ret[index] = v
					wg.Done()
					return
				case <-ctx.Done():
					wg.Done()
					return
				}
			}(item, i)
		}

		//wait for the iteration wait group to complete...
		wg.Wait()

		if iterProm.state == rejected {
			return
		}

		resolve(ret)

	})

	return p

}

func MapSeries(items []interface{}, iteratorFunc PromiseIteratorFunc) *Promise {
	p := New(func(resolve func(interface{}), reject func(error)) {

		ret := make([]interface{}, len(items))

		for i, item := range items {
			v, err := iteratorEstablishFunc(item, iteratorFunc)
			if err != nil {
				reject(err)
				return
			}
			ret[i] = v
		}

		resolve(ret)

	})

	p.isIterable = true

	return p

}

func Props(items map[string]interface{}, iteratorFunc PromiseIteratorFunc) *Promise {
	//in parallel run the iteratorFunc against the items and return a same-order array in the resolve function or error the wrapped promise

	p := newIterPromise(len(items), func(wg *sync.WaitGroup, iterProm *Promise, resolve func(interface{}), reject func(error)) {

		ctx, cancel := context.WithCancel(context.Background())
		//create the destination item...
		mutex := &sync.Mutex{}
		ret := make(map[string]interface{}, len(items))
		defer cancel()

		for key, item := range items {
			go func(inItem interface{}, inKey string) {
				vchan, echan := iteratorEstablishFuncChan(inItem, iteratorFunc)
				defer close(vchan)
				defer close(echan)
				select {
				case err := <-echan:
					if err != nil {
						cancel()
						reject(err)
						wg.Done()
						return
					}
				case v := <-vchan:
					mutex.Lock()
					defer mutex.Unlock()
					ret[inKey] = v
					wg.Done()
					return
				case <-ctx.Done():
					wg.Done()
					return
				}
			}(item, key)
		}

		//wait for the iteration wait group to complete...
		wg.Wait()

		if iterProm.state == rejected {
			return
		}

		resolve(ret)

	})

	return p

}
