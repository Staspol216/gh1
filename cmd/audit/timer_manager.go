package pvz_worker_audit

import "time"

type WorkerTimerManager struct {
	timer   *time.Timer
	timeout <-chan time.Time
}

func (wT *WorkerTimerManager) clear() {
	wT.timer.Stop()
	wT.timeout = nil
	wT.timer = nil
}

func (wT *WorkerTimerManager) start() {
	wT.timer = time.NewTimer(5 * time.Second)
	wT.timeout = wT.timer.C
}

func (wT *WorkerTimerManager) reset() {
	wT.timer.Reset(5 * time.Second)
	wT.timeout = wT.timer.C
}

func (wT *WorkerTimerManager) drain() {
	if wT.timer != nil && !wT.timer.Stop() {
		select {
		case <-wT.timer.C:
		default:
		}
	}
}

func (wT *WorkerTimerManager) isAlive() bool {
	return wT.timer != nil
}
