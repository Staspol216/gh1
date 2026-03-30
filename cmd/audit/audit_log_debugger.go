package pvz_worker_audit

import "fmt"

type AuditDebugger struct{}

func (wT *AuditDebugger) logWorkerStarted(index int) {
	fmt.Printf("Worker %d started\n", index)
}

func (wT *AuditDebugger) logWorkerFinished(index int) {
	fmt.Printf("Worker %d finished\n", index)
}

func (wT *AuditDebugger) logWorkerDoneJobAfterTimeout(index int) {
	fmt.Printf("Worker %d done the jobs after timeout\n", index)
}

func (wT *AuditDebugger) logWorkerTookJob(index int, id string) {
	fmt.Printf("Worker %d took the job %s\n", index, id)
}

func (wT *AuditDebugger) logWorkerDoneJobsAfterReachingTimeout(index int) {
	fmt.Printf("Worker %d done the jobs by reaching batch limit\n", index)
}

func (wT *AuditDebugger) logButchCapacity(number int) {
	fmt.Println("Batch capacity:", number)
}

func (wT *AuditDebugger) logOutputWorkerStarted(index int) {
	fmt.Printf("Output worker %d started\n", index)
}

func (wT *AuditDebugger) logOutputWorkerFinished(index int) {
	fmt.Printf("Output worker %d finished\n", index)
}

func (wT *AuditDebugger) logOutputWorkerGetJobForProcess(index int) {
	fmt.Printf("Output worker %d get job for process\n", index)
}

func (wT *AuditDebugger) logAuditLogRecordSaved() {
	fmt.Println("Audit log record saved")
}

type AuditDebuggerStub struct{}

func (wT *AuditDebuggerStub) logWorkerStarted(index int) {}

func (wT *AuditDebuggerStub) logWorkerFinished(index int) {}

func (wT *AuditDebuggerStub) logWorkerDoneJobAfterTimeout(index int) {}

func (wT *AuditDebuggerStub) logWorkerTookJob(index int, id string) {}

func (wT *AuditDebuggerStub) logWorkerDoneJobsAfterReachingTimeout(index int) {}

func (wT *AuditDebuggerStub) logButchCapacity(number int) {}
