package leader_worker_model

import (
	"context"
	"goroutine_pool/internal"
	"goroutine_pool/internal/models/leader_worker_model/role"
	"goroutine_pool/internal/timer/time_heap"
	"goroutine_pool/internal/types"
	"goroutine_pool/log_plus"
)

var _ internal.GoroutinePool = (*LeaderWorkerModel)(nil)

type LeaderWorkerModel struct {
	leader *role.Leader
}

func (lw *LeaderWorkerModel) Submit(ctx context.Context, work *types.Work) {
	log_plus.Printf(log_plus.DEBUG_LEADER, "submit work %d\n", work.ID)
	workList := types.GetWorkList()
	workList.M1.Lock()
	workList.UntouchedWorks = append(workList.UntouchedWorks, work)
	workList.M1.Unlock()
}

func (lw *LeaderWorkerModel) Result(ctx context.Context) (work *types.Work, has bool) {
	workList := types.GetWorkList()
	workList.M2.Lock()
	if len(workList.FinishedWorks) > 0 {
		has = true
		work = workList.FinishedWorks[0]
		workList.FinishedWorks = workList.FinishedWorks[1:]
	} else {
		has = false
	}
	workList.M2.Unlock()
	return
}

func (lw *LeaderWorkerModel) Init(ctx context.Context, idealWorkerCount int) {
	lw.leader = role.CreateAndInitLeader(idealWorkerCount, &time_heap.TimeHeap{})
}

func (lw *LeaderWorkerModel) Run(ctx context.Context) {
	lw.leader.Run(ctx)
}

func (lw *LeaderWorkerModel) Stop() {
	lw.leader.Stop()
}

func (lw *LeaderWorkerModel) ToString() string {
	return lw.leader.ToString()
}
