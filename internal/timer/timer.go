package timer

import "time"

/*
key = 0 : 使用自增key，如果已经存在该key的监听事件，报错
key > 0 : 使用指定传入的key，如果已经存在该key的监听事件，报错
key < 0 : 使用指定传入的key，如果已经存在该key的监听事件，更新
返回 : 本次使用的key
*/

type Timer interface {
	Init(bool)
	Register(time.Duration, func([]any) error, []any, int64) int64
	Remove(int64)
	Callback() error
	C() <-chan time.Time
}
