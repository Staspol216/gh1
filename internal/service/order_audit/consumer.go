package order_audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/Staspol216/gh1/pkg/tracing"
	"go.uber.org/zap"
)

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
				app_logger.MyLogger.Info("channel closed, exiting goroutine")
				return
			}
			monitoring.ObserveKafkaMessage("consume", nil)
			c.log(msg)
		case err, ok := <-c.Partition.Errors():
			if ok {
				app_logger.MyLogger.Error("kafka consumer error", zap.Error(err))
				monitoring.ObserveKafkaMessage("consume", err)
			}
		case <-c.Context.Done():
			app_logger.MyLogger.Info("reader finished by context done")
			return
		}
	}
}

func (c *OrderAuditLogPartitionConsumer) log(job *sarama.ConsumerMessage) {
	startTime := time.Now()
	span, _ := tracing.StartSpanFromContext(c.Context, "Kafka.ConsumeOrderAuditLog")
	span.SetTag("topic", job.Topic)
	span.SetTag("partition", job.Partition)
	span.SetTag("offset", job.Offset)

	var task order_outbox.OrderOutboxTask
	if err := json.Unmarshal(job.Value, &task); err != nil {
		app_logger.MyLogger.Error("failed to unmarshal order audit log", zap.Error(err))
		monitoring.ObserveKafkaMessage("unmarshal", err)
		tracing.FinishSpan(span, startTime, err)
	} else {
		monitoring.ObserveKafkaMessage("unmarshal", nil)
		span.SetTag("task_id", task.ID)
		span.SetTag("order_status", string(task.OrderStatus))
		app_logger.MyLogger.Info("audit log record",
			zap.Int64("task_id", task.ID),
			zap.String("status", task.Status),
			zap.String("order_status", string(task.OrderStatus)),
			zap.String("description", task.Description),
			zap.Time("timestamp", task.Timestamp),
		)
		tracing.FinishSpan(span, startTime, nil)
	}
}
