package role

import (
	"context"
	"errors"
	"fmt"
	"goroutine_pool/internal/timer"
	"goroutine_pool/internal/types"
	"goroutine_pool/log_plus"
	"goroutine_pool/monitor"
	"sync/atomic"
	"time"
)

const (
	MAXGOROUTINE int = 1e3
)

type Leader struct {
	untouchedTask    []*types.Task      // 存储所有未完成的task列表
	finishedTask     []*types.Task      // 存储所有的已完成的task列表
	idealWorkerCount int                // 理想的工作线程数量
	workers          map[int64]*worker  // 所有的worker
	idleWorkers      []int64            // 闲置的worker
	deadWorkers      map[int64]struct{} // 死亡的worker
	runningWorkers   map[int64]struct{} // 正在运行的worker
	workerTick       chan int64         // worker完成task回复
	myTick           chan struct{}      // 协程池变化信号，轮询、有新的idle的worker产生
	timer            timer.Timer        // 计时器
	stopping         atomic.Bool        // 准备停止
	stopped          chan struct{}      // 停止信号
}

func CreateAndInitLeader(idealWorkerCount int, timer timer.Timer) *Leader {
	l := &Leader{
		untouchedTask:    []*types.Task{},
		finishedTask:     []*types.Task{},
		idealWorkerCount: min(max(idealWorkerCount, 1), MAXGOROUTINE),
		workers:          map[int64]*worker{},
		idleWorkers:      []int64{},
		deadWorkers:      map[int64]struct{}{},
		runningWorkers:   map[int64]struct{}{},
		workerTick:       make(chan int64, MAXGOROUTINE),
		myTick:           make(chan struct{}, 1),
		timer:            timer,
		stopping:         atomic.Bool{},
		stopped:          make(chan struct{}),
	}
	l.stopping.Store(false)
	l.timer.Init(false)
	l.tick()
	log_plus.Printf(log_plus.DEBUG_LEADER, "leader: init over\n")
	return l
}

func (l *Leader) Run(ctx context.Context) {
	_ctx, cancel := context.WithCancel(ctx)
LOOP:
	for {
		monitor.AddLoops()
		if monitor.ReturnLoops()%1000 == 0 {
			log_plus.Println(log_plus.Grade, l.ToString())
		}
		if l.stopping.Load() && len(l.runningWorkers) == 0 && len(l.untouchedTask) == 0 && len(l.finishedTask) == 0 {
			break LOOP
		}
		select {
		case workerId, open := <-l.workerTick:
			if !open {
				panic("leader: closed workerTick")
			}
			if err := l.receive(workerId); err != nil {
				panic(err)
			}
		case _, open := <-l.myTick:
			if !open {
				panic("leader: closed myTick")
			}
			l.expand(_ctx, l.expendCount())
			l.pullWorks()
			if err := l.distribute(); err != nil {
				panic(err)
			}
			l.pushWorks()
		case <-l.timer.C():
			if err := l.timer.Callback(); err != nil {
				panic(err)
			}
		case <-ctx.Done():
			break LOOP
		}
	}
	close(l.stopped)
	log_plus.Println(log_plus.DEBUG_LEADER, "leader: stopped")
	cancel()
}

func (l *Leader) Stop() {
	l.stopping.Store(true)
	<-l.stopped
}

func (l *Leader) distribute() error {
	for len(l.idleWorkers) > 0 && len(l.untouchedTask) > 0 {
		monitor.AddLoops()
		workerId := l.idleWorkers[0]
		l.idleWorkers = l.idleWorkers[1:]
		l.runningWorkers[workerId] = struct{}{}
		task := l.untouchedTask[0]
		l.untouchedTask = l.untouchedTask[1:]
		task.WorkerID = workerId
		select {
		case l.workers[workerId].distributeChan <- task:
		default:
			return errors.New(fmt.Sprintf("leader: block on worker %d's distributeChan", workerId))
		}
		l.timer.Register(task.Work.Timeout, func(params []any) error {
			task, ok := params[0].(*types.Task)
			if !ok {
				return errors.New("leader: task type error")
			}
			l.deadWorkers[task.WorkerID] = struct{}{}
			delete(l.runningWorkers, task.WorkerID)
			task.Reset()
			if task.Retry == 0 {
				task.Success = false
				l.finishedTask = append(l.finishedTask, task)
				log_plus.Printf(log_plus.DEBUG_LEADER, "leader: task %d timeout, work %d failed", task.ID, task.Work.ID)
			} else {
				l.untouchedTask = append(l.untouchedTask, task)
				log_plus.Printf(log_plus.DEBUG_LEADER, "leader: task %d timeout, retry times %d\n", task.ID, task.Retry)
			}
			l.tick()
			return nil
		}, []any{task}, int64(task.ID))
		log_plus.Printf(log_plus.DEBUG_LEADER, "leader: distribute task %d to worker %d\n", task.ID, workerId)
	}
	return nil
}

func (l *Leader) receive(workerId int64) error {
	if _, has := l.runningWorkers[workerId]; !has {
		if _, has = l.deadWorkers[workerId]; has {
			delete(l.deadWorkers, workerId)
			log_plus.Printf(log_plus.DEBUG_LEADER, "leader: refuse worker %d's task %d\n", workerId, l.workers[workerId].getFinishedTask().ID)
		} else {
			return errors.New("leader: ghost worker")
		}
	} else {
		task := l.workers[workerId].getFinishedTask()
		task.Success = true
		l.finishedTask = append(l.finishedTask, task)
		delete(l.runningWorkers, workerId)
		l.timer.Remove(int64(task.ID))
		log_plus.Printf(log_plus.DEBUG_LEADER, "leader: worker %d process task %d successfully\n", workerId, task.ID)
	}
	l.idleWorkers = append(l.idleWorkers, workerId)
	l.tick()
	return nil
}

func (l *Leader) pullWorks() {
	workList := types.GetWorkList()
	workList.M1.Lock()
	works := workList.UntouchedWorks
	workList.UntouchedWorks = []*types.Work{}
	workList.M1.Unlock()
	if len(works) > 0 {
		for _, v := range works {
			monitor.AddLoops()
			l.untouchedTask = append(l.untouchedTask, types.Bind(v))
		}
		l.tick()
	}
}

func (l *Leader) pushWorks() {
	var works []*types.Work
	for _, v := range l.finishedTask {
		monitor.AddLoops()
		works = append(works, v.Unbind())
	}
	l.finishedTask = []*types.Task{}
	if len(works) > 0 {
		workList := types.GetWorkList()
		workList.M2.Lock()
		workList.FinishedWorks = append(workList.FinishedWorks, works...)
		workList.M2.Unlock()
	} else {
		l.timer.Register(time.Second, func([]any) error {
			l.tick()
			return nil
		}, nil, -1)
	}
}

func (l *Leader) expand(ctx context.Context, n int) {
	if n > 0 {
		for i := 0; i < n; i++ {
			monitor.AddLoops()
			worker := createAndInitWorker(l.workerTick)
			l.workers[worker.id] = worker
			l.idleWorkers = append(l.idleWorkers, worker.id)
			go worker.run(ctx)
		}
		log_plus.Printf(log_plus.DEBUG_LEADER, "leader: add %d workers\n", n)
	}
}

func (l *Leader) expendCount() int {
	if len(l.idleWorkers) == 0 && len(l.runningWorkers) < l.idealWorkerCount && l.idealWorkerCount-len(l.runningWorkers)+len(l.workers) < MAXGOROUTINE && len(l.untouchedTask) > 0 {
		return l.idealWorkerCount - len(l.runningWorkers)
	}
	return 0
}

func (l *Leader) tick() {
	select {
	case l.myTick <- struct{}{}:
	default:
	}
}

func (l *Leader) ToString() string {
	ret := fmt.Sprintf("=========================== leader ===========================\n")
	ret += fmt.Sprintf("		worker count: %d\n", len(l.workers))
	var tmp []int64
	for k := range l.runningWorkers {
		tmp = append(tmp, k)
	}
	ret += fmt.Sprintf("		running workers: %v\n", tmp)
	ret += fmt.Sprintf("		idle workers: %v\n", l.idleWorkers)
	tmp = []int64{}
	for k := range l.deadWorkers {
		tmp = append(tmp, k)
	}
	ret += fmt.Sprintf("		dead workers: %v\n", tmp)
	ret += fmt.Sprintf("		untouchedTask count: %d\n", len(l.untouchedTask))
	ret += fmt.Sprintf("		finishedTasks count: %d\n", len(l.finishedTask))
	ret += fmt.Sprintf("==============================================================")
	return ret
}
