package order_audit

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
				app_logger.MyLogger.Info("tasks channel closed, exiting writer")
				return
			}

			err := w.work(tasksBatch)

			if err != nil {
				app_logger.MyLogger.Error("order audit producer work failed", zap.Error(err))
			}

			app_logger.MyLogger.Info("worker processed jobs", zap.Int("tasks_count", len(tasksBatch)))
			continue
		case <-w.Context.Done():
			app_logger.MyLogger.Info("writer was finished by context done")
			return
		}
	}
}

func (w *OrderAuditLogProducer) work(tasks []order_outbox.OrderOutboxTask) error {

	for _, task := range tasks {

		requestID := uuid.New().String()
		bytes, err := json.Marshal(task)
		if err != nil {
			app_logger.MyLogger.Error("failed to marshal JSON for task", zap.Int64("task_id", task.ID), zap.Error(err))
			monitoring.ObserveKafkaMessage("marshal", err)
			continue
		}
		monitoring.ObserveKafkaMessage("marshal", nil)

		msg := &sarama.ProducerMessage{
			Topic: "order_audit_logs",
			Key:   sarama.StringEncoder(requestID),
			Value: sarama.ByteEncoder(bytes),
		}

		partition, offset, err := w.Producer.SendMessage(msg)

		if err != nil {
			app_logger.MyLogger.Error("failed to send message to Kafka", zap.Int64("task_id", task.ID), zap.Error(err))
			monitoring.ObserveKafkaMessage("produce", err)
			if ferr := w.Outbox.MarkTaskAsFailed(w.Context, task.ID); ferr != nil {
				app_logger.MyLogger.Error("failed to mark task as failed", zap.Int64("task_id", task.ID), zap.Error(ferr))
			}
			continue
		}
		monitoring.ObserveKafkaMessage("produce", nil)

		if ferr := w.Outbox.DeleteTask(w.Context, task.ID); ferr != nil {
			app_logger.MyLogger.Error("failed to delete task after successful send",
				zap.Int64("task_id", task.ID),
				zap.Int32("partition", partition),
				zap.Int64("offset", offset),
				zap.Error(ferr),
			)
			continue
		}
		monitoring.ObserveKafkaMessage("ack", nil)
	}

	return nil
}
