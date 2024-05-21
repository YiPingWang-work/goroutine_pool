package types

import "time"

type Task struct {
	ID         uint64 // 任务ID
	WorkerID   int64  // 分配的worker，未分配为-1
	Work       *Work  // 所属的work
	Success    bool   // 本次任务是否执行成功
	TmpParams  []any  // 本次任务的入参
	TmpReturns []any  // 本次任务的出参
	Retry      uint64 // 还剩下的重试次数
}

var (
	id uint64
)

func Bind(work *Work) *Task {
	id++
	work.BeginTime = time.Now()
	work.Success = false
	return &Task{
		ID:         id,
		WorkerID:   -1,
		Work:       work,
		Retry:      work.Retry,
		TmpParams:  work.Params,
		TmpReturns: work.Returns,
	}
}

func (t *Task) Reset() {
	t.WorkerID = -1
	t.Success = false
	t.TmpParams = t.Work.Params
	t.TmpReturns = t.Work.Returns
	t.Retry--
}

func (t *Task) Unbind() *Work {
	t.Work.Params = t.TmpParams
	t.Work.Returns = t.TmpReturns
	t.Work.Timestamp.EndTime = time.Now()
	t.Work.Success = t.Success
	return t.Work
}
