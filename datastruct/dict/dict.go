package dict

type Dict interface {
	Get(key string) (val interface{}, exist bool)
	Len()
	Put(key string, val interface{}) (result int)
	PutIfAbsent(key string, val interface{}) (result int)
	PutIfExist(key string, val interface{}) (result int)
}
