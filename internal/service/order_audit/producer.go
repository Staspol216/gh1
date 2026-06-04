package order_audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/google/uuid"
)

type OrderAuditLogProducer struct {
	Context  context.Context
	Producer sarama.SyncProducer
	Tasks    <-chan []order_outbox.OrderOutboxTask
	Outbox   pvz_order_service.Outbox
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

func (w *OrderAuditLogProducer) work(tasks []order_outbox.OrderOutboxTask) error {

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
