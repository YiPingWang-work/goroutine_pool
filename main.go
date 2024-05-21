package main

import (
	"context"
	"goroutine_pool/internal/models/leader_worker_model"
	"goroutine_pool/internal/types"
	"goroutine_pool/log_plus"
	"goroutine_pool/monitor"
	"math/rand"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log_plus.Grade = log_plus.DEBUG_OTHER | log_plus.DEBUG_LEADER | log_plus.DEBUG_TIME_HEAP
	log_plus.LogFile = "./log1"
	lw := &leader_worker_model.LeaderWorkerModel{}
	lw.Init(ctx, 10)
	loop := 5
	go func() {
		id := uint64(0)
		for i := 0; i < 100; i++ {
			if i == 50 {
				loop = 100
			}
			time.Sleep(5 * time.Second)
			for i := 0; i < loop; i++ {
				id++
				lw.Submit(ctx, &types.Work{
					ID:      id,
					Timeout: time.Second,
					Retry:   3,
					Fn: func(params []any) []any {
						time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000)))
						return params
					},
					Params:    []any{id},
					Returns:   []any{},
					Success:   false,
					Extra:     "",
					Timestamp: types.Timestamp{},
				})
			}
		}
		time.Sleep(time.Second * 120)
		lw.Stop()
	}()
	lw.Run(ctx)
	i := 0
	for {
		i++
		res, has := lw.Result(ctx)
		if !has {
			break
		} else {
			log_plus.Printf(log_plus.Grade, "%d success: %v, res: %v\n", res.ID, res.Success, res.Returns)
		}
	}
	cancel()
	log_plus.Println(log_plus.Grade, lw.ToString())
	log_plus.Println(log_plus.Grade, i)
	log_plus.Println(log_plus.Grade, monitor.ReturnLoops())
}
