package promise

func iteratorEstablishFuncChan(item interface{}, iteratorFunc PromiseIteratorFunc) (chan interface{}, chan error) {

	vchan := make(chan interface{})
	echan := make(chan error)
	go func() {
		switch item.(type) {
		case *Promise:
			p := item.(*Promise)
			pv, perr := wrapPromise(p)
			if perr != nil {
				echan <- perr
			} else {
				v, err := checkReturnedIteratorValue(iteratorFunc(pv))
				if err != nil {
					echan <- err
				} else {
					vchan <- v
				}
			}
		case Promise:
			p := item.(Promise)
			pv, perr := wrapPromise(&p)
			if perr != nil {
				echan <- perr
			} else {
				v, err := checkReturnedIteratorValue(iteratorFunc(pv))
				if err != nil {
					echan <- err
				} else {
					vchan <- v
				}
			}
		default:
			v, err := checkReturnedIteratorValue(iteratorFunc(item))
			if err != nil {
				echan <- err
			} else {
				vchan <- v
			}
		}
	}()
	return vchan, echan
}

func iteratorEstablishFunc(item interface{}, iteratorFunc PromiseIteratorFunc) (interface{}, error) {

	switch item.(type) {
	case *Promise:
		p := item.(*Promise)
		v, err := wrapPromise(p)
		if err != nil {
			return nil, err
		}
		return checkReturnedIteratorValue(iteratorFunc(v))
	case Promise:
		p := item.(Promise)
		v, err := wrapPromise(&p)
		if err != nil {
			return nil, err
		}
		return checkReturnedIteratorValue(iteratorFunc(v))
	default:
		return checkReturnedIteratorValue(iteratorFunc(item))
	}

}

func (promise *Promise) checkReturnedFulfilmentValue(retVal interface{}) (interface{}, error) {

	switch retVal.(type) {
	case *Promise:
		p := retVal.(*Promise)
		v, err := wrapPromise(p)
		if err != nil {
			return nil, err
		}
		return v, nil
	case Promise:
		p := retVal.(Promise)
		v, err := wrapPromise(&p)
		if err != nil {
			return nil, err
		}
		return v, nil
	default:
		return retVal, nil
	}

}

func (promise *Promise) checkReturnedRejectionValue(retVal interface{}, retErr error) (interface{}, error, bool) {

	switch retVal.(type) {
	case *Promise:
		p := retVal.(*Promise)
		v, err := wrapPromise(p)
		if err != nil {
			return nil, err, true
		}
		return v, nil, true
	case Promise:
		p := retVal.(Promise)
		v, err := wrapPromise(&p)
		if err != nil {
			return nil, err, true
		}
		return v, nil, true
	default:
		return retVal, retErr, false
	}

}

func checkReturnedIteratorValue(retVal interface{}, err error) (interface{}, error) {

	if err != nil {
		return nil, err
	}

	switch retVal.(type) {
	case *Promise:
		p := retVal.(*Promise)
		return wrapPromise(p)
	case Promise:
		p := retVal.(Promise)
		return wrapPromise(&p)
	default:
		return retVal, nil
	}

}

func wrapPromise(inPromise *Promise) (interface{}, error) {

	v, err := inPromise.Yield()

	return checkReturnedIteratorValue(v, err)

}
