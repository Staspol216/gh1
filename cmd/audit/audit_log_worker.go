package pvz_worker_audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

type AuditLogRepo interface {
	GetNewestUnprocessedTask(ctx context.Context) (*pvz_domain.OrderOutboxTask, error)
	MarkTaskAsProcessing(ctx context.Context, id int64) error
	MarkTaskAsFailed(ctx context.Context, id int64) error
	DeleteTasks(ctx context.Context, ids []int64) error
	DeleteTask(ctx context.Context, id int64) error
}

type OrderAuditLogPartitionWriter struct {
	Context    context.Context
	Producer   sarama.SyncProducer
	Tasks      <-chan *pvz_domain.OrderOutboxTask
	OutboxRepo AuditLogRepo
	Debugger   *AuditDebugger
}

func (w *OrderAuditLogPartitionWriter) Run() {

	tm := &WorkerTimerManager{}

	const batchCapacity = 5
	batch := make([]*pvz_domain.OrderOutboxTask, 0, batchCapacity)

	w.Debugger.logWorkerStarted(1)

	for {
		select {
		case task, ok := <-w.Tasks:
			if !ok {
				log.Println("Tasks channel closed, exiting writer")
				return
			}
			w.Debugger.logWorkerTookJob(1, task.RequestID)
			batch = append(batch, task)
			w.Debugger.logButchCapacity(len(batch))

			if len(batch) >= batchCapacity {
				if tm.isAlive() {
					tm.clear()
				}
				batch = w.work(batch)
				w.Debugger.logWorkerDoneJobsAfterReachingTimeout(1)
				continue
			}

			if !tm.isAlive() {
				tm.start()
			} else {
				tm.drain()
				tm.reset()
			}
		case <-tm.timeout:
			w.Debugger.logWorkerDoneJobAfterTimeout(1)
			batch = w.work(batch)
			tm.clear()
		case <-w.Context.Done():
			log.Printf("Writer was finished by context done")
			return
		}
	}
}

func (w *OrderAuditLogPartitionWriter) work(tasks []*pvz_domain.OrderOutboxTask) []*pvz_domain.OrderOutboxTask {

	count := 0

	messages := make([]*sarama.ProducerMessage, 0, len(tasks))
	taskIDMap := make(map[*sarama.ProducerMessage]*pvz_domain.OrderOutboxTask)

	for _, task := range tasks {

		requestID := uuid.New().String()
		bytes, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal JSON")
			continue
		}

		msg := &sarama.ProducerMessage{
			Topic: "order_audit_logs",
			Key:   sarama.StringEncoder(requestID),
			Value: sarama.ByteEncoder(bytes),
		}

		messages = append(messages, msg)
		taskIDMap[msg] = task

		count++
	}

	maxRetries := 3
	var err error
	for i := range maxRetries {
		if w.Context.Err() != nil {
			log.Printf("Context cancelled, aborting batch send")
			return tasks
		}
		err = w.Producer.SendMessages(messages)
		if err == nil {
			ids := make([]int64, len(tasks))

			for _, task := range tasks {
				ids = append(ids, *task.ID)
			}

			w.OutboxRepo.DeleteTasks(w.Context, ids)
			return tasks[count:]
		}
		// If some messages failed, Sarama returns a ProducerErrors slice
		producerErrors, ok := err.(sarama.ProducerErrors)
		if ok {
			for _, perr := range producerErrors {
				log.Printf("Retry %d: Failed to send message to Kafka: %v", i+1, perr.Err)
			}
		} else {
			log.Printf("Retry %d: Failed to send batch to Kafka: %v", i+1, err)
		}
	}

	producerErrors, ok := err.(sarama.ProducerErrors)
	if ok {
		for _, perr := range producerErrors {
			task, found := taskIDMap[perr.Msg]
			if found {
				if ferr := w.OutboxRepo.MarkTaskAsFailed(w.Context, *task.ID); ferr != nil {
					log.Printf("MarkTaskAsFailed for task %d: %v", *task.ID, ferr)
				}
			}
		}
	} else {
		// If not a ProducerErrors, mark all as failed
		for _, msg := range messages {
			task := taskIDMap[msg]
			if ferr := w.OutboxRepo.MarkTaskAsFailed(w.Context, *task.ID); ferr != nil {
				log.Printf("MarkTaskAsFailed for task %d: %v", *task.ID, ferr)
			}
		}
	}

	return tasks[count:]
}

type OrderAuditLogPartitionReader struct {
	Context   context.Context
	Partition sarama.PartitionConsumer
}

func (r *OrderAuditLogPartitionReader) Run() {
	for {
		select {
		// Чтение сообщения из Kafka
		case msg, ok := <-r.Partition.Messages():
			if !ok {
				log.Println("Channel closed, exiting goroutine")
				return
			}
			r.log(msg)
		case err, ok := <-r.Partition.Errors():
			if ok {
				log.Printf("Kafka consumer error: %v", err)
			}
		case <-r.Context.Done():
			log.Printf("Reader finished by context done")
			return
		}
	}
}

func (w *OrderAuditLogPartitionReader) log(job *sarama.ConsumerMessage) {
	fmt.Println("----------- AUDIT LOG RECORD -----------")
	var task pvz_domain.OrderOutboxTask
	if err := json.Unmarshal(job.Value, &task); err != nil {
		fmt.Printf("Failed to unmarshal audit log: %v\n", err)
	} else {
		spew.Dump(task)
	}
	fmt.Println("----------------------------------------")
}
