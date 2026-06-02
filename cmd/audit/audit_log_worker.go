package pvz_worker_audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

type Outbox interface {
	LockPending(ctx context.Context) ([]pvz_domain.OrderOutboxTask, error)
	MarkTaskAsFailed(ctx context.Context, id int64) error
	DeleteTasks(ctx context.Context, ids []int64) error
	DeleteTask(ctx context.Context, id int64) error
}

type OrderAuditLogProducer struct {
	Context  context.Context
	Producer sarama.SyncProducer
	Tasks    <-chan []pvz_domain.OrderOutboxTask
	Outbox   Outbox
}

func (w *OrderAuditLogProducer) Run() {

	for {
		select {
		case tasksBatch, ok := <-w.Tasks:
			if !ok {
				log.Println("Tasks channel closed, exiting writer")
				return
			}

			err := w.work(tasksBatch)

			if err != nil {
				log.Println(fmt.Errorf("w.work: %w", err))
			}

			fmt.Println("Worker process jobs")
			continue
		case <-w.Context.Done():
			log.Printf("Writer was finished by context done")
			return
		}
	}
}

func (w *OrderAuditLogProducer) work(tasks []pvz_domain.OrderOutboxTask) error {

	for _, task := range tasks {

		requestID := uuid.New().String()
		bytes, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal JSON for task %d: %v", task.ID, err)
			continue
		}

		msg := &sarama.ProducerMessage{
			Topic: "order_audit_logs",
			Key:   sarama.StringEncoder(requestID),
			Value: sarama.ByteEncoder(bytes),
		}

		partition, offset, err := w.Producer.SendMessage(msg)

		if err != nil {
			log.Printf("Failed to send message to Kafka for task %d: %v", task.ID, err)
			if ferr := w.Outbox.MarkTaskAsFailed(w.Context, task.ID); ferr != nil {
				log.Printf("MarkTaskAsFailed for task %d: %v", task.ID, ferr)
			}
			continue
		}

		if ferr := w.Outbox.DeleteTask(w.Context, task.ID); ferr != nil {
			log.Printf("Failed to delete task %d after successful send (partition: %d, offset: %d): %v", task.ID, partition, offset, ferr)
			continue
		}
	}

	return nil
}

type OrderAuditLogPartitionConsumer struct {
	Context   context.Context
	Partition sarama.PartitionConsumer
}

func (c *OrderAuditLogPartitionConsumer) Run() {
	for {
		select {
		// Чтение сообщения из Kafka
		case msg, ok := <-c.Partition.Messages():
			if !ok {
				log.Println("Channel closed, exiting goroutine")
				return
			}
			c.log(msg)
		case err, ok := <-c.Partition.Errors():
			if ok {
				log.Printf("Kafka consumer error: %v", err)
			}
		case <-c.Context.Done():
			log.Printf("Reader finished by context done")
			return
		}
	}
}

func (c *OrderAuditLogPartitionConsumer) log(job *sarama.ConsumerMessage) {
	fmt.Println("----------- AUDIT LOG RECORD -----------")
	var task pvz_domain.OrderOutboxTask
	if err := json.Unmarshal(job.Value, &task); err != nil {
		fmt.Printf("Failed to unmarshal audit log: %v\n", err)
	} else {
		spew.Dump(task)
	}
	fmt.Println("----------------------------------------")
}
