package types

import (
	"sync"
	"time"
)

type Work struct {
	ID        uint64            // 工作ID
	Timeout   time.Duration     // 执行一次这个工作的最高延时
	Retry     uint64            // 允许重试的次数
	Fn        func([]any) []any // 执行函数
	Params    []any             // 执行参数（传入传出参数）
	Returns   []any             // 执行返回（传出参数）
	Success   bool              // 执行是否成功（传出参数）
	Extra     string            // 执行的补充信息（传出参数）
	Timestamp                   // 整体执行的时间戳（传出参数）
}

type WorkList struct {
	UntouchedWorks []*Work
	FinishedWorks  []*Work
	M1             sync.Mutex
	M2             sync.Mutex
}

var (
	workListInst *WorkList
	workListOnce sync.Once
)

func GetWorkList() *WorkList {
	workListOnce.Do(func() {
		workListInst = &WorkList{
			UntouchedWorks: []*Work{},
			FinishedWorks:  []*Work{},
		}
	})
	return workListInst
}
