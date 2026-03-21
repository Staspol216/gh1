package pvz_worker_audit

import (
	"context"
	"fmt"
	"sync"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/audit_log"
)

type ProcessStrategy = string

const (
	Print    ProcessStrategy = "print"
	SaveToDB ProcessStrategy = "saveToDB"
)

type AuditLogRepo interface {
	AddAuditLog(audit_log *pvz_domain.AuditLog) (int64, error)
}

type Worker struct {
	ProcessStrategy ProcessStrategy
	Context         context.Context
	In              <-chan *pvz_domain.AuditLog
	Out             chan *pvz_domain.AuditLog
	Wg              *sync.WaitGroup
	Repo            AuditLogRepo
	Debugger        *AuditDebugger
}

func (w *Worker) RunAndServe(index int) {
	w.Wg.Add(2)

	go w.Run(index)
	go w.Serve(index)
}

func (w *Worker) Run(index int) {
	defer w.Wg.Done()

	tm := &WorkerTimerManager{}

	const batchCapacity = 5
	batch := make([]*pvz_domain.AuditLog, 0, batchCapacity)

	w.Debugger.logWorkerStarted(index)

	for {
		select {
		case <-w.Context.Done():
			if tm.isAlive() {
				tm.clear()
			}
			w.work(batch)
			w.Debugger.logWorkerFinished(index)
			return
		case <-tm.timeout:
			w.Debugger.logWorkerDoneJobAfterTimeout(index)
			batch = w.work(batch)
			tm.clear()
		case job := <-w.In:
			w.Debugger.logWorkerTookJob(index, job.RequestID)

			batch = append(batch, job)
			w.Debugger.logButchCapacity(len(batch))

			if len(batch) >= batchCapacity {
				if tm.isAlive() {
					tm.clear()
				}
				batch = w.work(batch)
				w.Debugger.logWorkerDoneJobsAfterReachingTimeout(index)
				continue
			}

			if !tm.isAlive() {
				tm.start()
			} else {
				tm.drain()
				tm.reset()
			}
		}
	}
}

func (w *Worker) Serve(index int) {
	defer w.Wg.Done()

	w.Debugger.logOutputWorkerStarted(index)

	for {
		select {
		case <-w.Context.Done():
			w.Debugger.logOutputWorkerFinished(index)
			return
		case j := <-w.Out:
			w.Debugger.logOutputWorkerGetJobForProcess(index)
			w.proccess(j)
		}
	}
}

func (w *Worker) proccess(job *pvz_domain.AuditLog) {
	switch w.ProcessStrategy {
	case Print:
		w.printLog(job)
	case SaveToDB:
		w.saveLog(job)
	}
}

func (w *Worker) printLog(job *pvz_domain.AuditLog) {
	fmt.Println("----------- AUDIT LOG RECORD -----------")
	fmt.Printf("%#v\n", job)
	fmt.Println("----------------------------------------")
}

func (w *Worker) saveLog(job *pvz_domain.AuditLog) {
	_, err := w.Repo.AddAuditLog((*pvz_domain.AuditLog)(job))
	if err != nil {
		return
	}
	w.Debugger.logAuditLogRecordSaved()
}

func (w *Worker) do(job *pvz_domain.AuditLog) {
	w.Out <- job
}

func (w *Worker) work(batch []*pvz_domain.AuditLog) []*pvz_domain.AuditLog {
	count := 0

	for _, job := range batch {
		w.do(job)
		count++
	}

	return batch[count:]
}
