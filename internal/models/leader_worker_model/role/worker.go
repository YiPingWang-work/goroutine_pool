package role

import (
	"context"
	"fmt"
	"goroutine_pool/internal/types"
	"goroutine_pool/log_plus"
)

type worker struct {
	id             int64            // worker的唯一ID
	distributeChan chan *types.Task // 获取任务的管道
	finishedTask   *types.Task      // 完成的任务的存储位置
	workerTick     chan<- int64     // 完成任务后回复信号
}

var (
	autoId int64 = 0
)

func createAndInitWorker(workerTick chan<- int64) *worker {
	autoId++
	log_plus.Printf(log_plus.DEBUG_WORKER, "worker %d: created\n", autoId)
	return &worker{
		id:             autoId,
		distributeChan: make(chan *types.Task, 1),
		finishedTask:   nil,
		workerTick:     workerTick,
	}
}

func (w *worker) run(ctx context.Context) {
	for {
		select {
		case task, open := <-w.distributeChan:
			if !open {
				panic(fmt.Sprintf("worker %d: closed DistributeChan", w.id))
			}
			log_plus.Printf(log_plus.DEBUG_WORKER, "worker %d: process task %d of work %d\n", w.id, task.ID, task.Work.ID)
			task.TmpReturns = task.Work.Fn(task.TmpParams)
			w.finishedTask = task
			w.workerTick <- w.id
			log_plus.Printf(log_plus.DEBUG_WORKER, "worker %d: finish task %d of work %d\n", w.id, task.ID, task.Work.ID)
		case <-ctx.Done():
			log_plus.Printf(log_plus.DEBUG_WORKER, "worker %d: stopped\n", w.id)
			return
		}
	}
}

func (w *worker) getFinishedTask() *types.Task {
	return w.finishedTask
}
