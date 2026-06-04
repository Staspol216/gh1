package order_audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/davecgh/go-spew/spew"
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
	var task order_outbox.OrderOutboxTask
	if err := json.Unmarshal(job.Value, &task); err != nil {
		fmt.Printf("Failed to unmarshal order_audit log: %v\n", err)
	} else {
		spew.Dump(task)
	}
	fmt.Println("----------------------------------------")
}
