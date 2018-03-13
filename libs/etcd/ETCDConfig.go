package ETCD

import (
	"reflect"
	"sync"

)
type ETCDConfig struct {
	Key     string
	Type    reflect.Type
	Content interface{}
	Locker  sync.RWMutex
}
