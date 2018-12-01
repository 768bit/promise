package promise

type PromiseExecutor = func(resolve func(interface{}), reject func(error))
type PromiseIteratorFunc = func(item interface{}) (interface{}, error)
