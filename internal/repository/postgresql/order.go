package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/Staspol216/gh1/internal/db"
	pvz_model "github.com/Staspol216/gh1/internal/models/order"
)

type OrderRepo struct {
	db db.DB
}

func NewOrderRepo(database db.DB, ctx context.Context) (*OrderRepo, error) {
	return &OrderRepo{
		db: database,
	}, nil
}

func (r *OrderRepo) Add(ctx context.Context, order *pvz_model.Order) (int64, error) {
	query := `INSERT INTO orders (
		recipient_id,
		expiration_date,
		status,
		weight,
		worth
	) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	row := r.db.ExecQueryRow(ctx, query,
		order.RecipientID,
		order.ExpirationDate,
		order.Status,
		order.Weight,
		order.Worth,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		log.Println(err)
	}
	return id, err
}

func (r *OrderRepo) AddHistoryRecord(ctx context.Context, record *pvz_model.OrderRecord, orderId int64) (int64, error) {
	query := `INSERT INTO order_records (
		order_id,
		description,
		timestamp,
		status
	) VALUES ($1, $2, $3, $4) RETURNING id;`

	row := r.db.ExecQueryRow(ctx, query,
		orderId,
		record.Description,
		record.Timestamp,
		record.Status,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		log.Println(err)
	}
	return id, err
}

func (r *OrderRepo) Delete(ctx context.Context, orderId int64) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM orders WHERE ID = $1;`, orderId)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("no row found to delete")
	}
	return nil
}

func (r *OrderRepo) Update(ctx context.Context, updatedOrder *pvz_model.Order) error {
	query := `UPDATE orders SET recipient_id=$1, expiration_date=$2, delivered_date=$3, refunded_date=$4, returned_date=$5, status=$6, weight=$7, worth=$8 WHERE id=$9 RETURNING id;`

	row := r.db.ExecQueryRow(ctx, query,
		updatedOrder.RecipientID,
		updatedOrder.ExpirationDate,
		updatedOrder.DeliveredDate,
		updatedOrder.RefundedDate,
		updatedOrder.ReturnedDate,
		updatedOrder.Status,
		updatedOrder.Weight,
		updatedOrder.Worth,
		updatedOrder.ID,
	)

	var updatedID int64
	if err := row.Scan(&updatedID); err != nil {
		return err
	}

	return nil
}

func (r *OrderRepo) GetAll(ctx context.Context) ([]*pvz_model.Order, error) {
	var orderDTOs []orderDTO
	err := r.db.Select(ctx, &orderDTOs, `SELECT * FROM orders ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}

	var recordDTOs []orderRecordDTO
	orderRecordsErr := r.db.Select(ctx, &recordDTOs, `
        SELECT id, order_id, timestamp, status, description
        FROM order_records
        ORDER BY order_id, timestamp
    `)
	if orderRecordsErr != nil {
		return nil, orderRecordsErr
	}

	m := make(map[int64][]pvz_model.OrderRecord)
	for _, recordDTO := range recordDTOs {
		orderRecordModel := transformOrderRecordDtoToModel(&recordDTO)
		m[recordDTO.OrderID] = append(m[recordDTO.OrderID], *orderRecordModel)
	}

	var orders []*pvz_model.Order

	for _, dto := range orderDTOs {
		orderModel := transformOrderDtoToModel(&dto)
		orderModel.History = m[orderModel.ID]
		orders = append(orders, orderModel)
	}

	return orders, nil
}

func (r *OrderRepo) GetList(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error) {

	var orderDTOs []orderDTO
	err := r.db.Select(ctx, &orderDTOs, `
		SELECT * 
		FROM orders
		ORDER BY id ASC
		LIMIT $1
		OFFSET $2
	`, pagination.Limit, pagination.Offset)

	if err != nil {
		return nil, err
	}

	var recordDTOs []orderRecordDTO
	orderRecordsErr := r.db.Select(ctx, &recordDTOs, `
        SELECT id, order_id, timestamp, status, description
        FROM order_records
        ORDER BY order_id, timestamp
    `)

	if orderRecordsErr != nil {
		return nil, orderRecordsErr
	}

	m := make(map[int64][]pvz_model.OrderRecord)
	for _, recordDTO := range recordDTOs {
		orderRecordModel := transformOrderRecordDtoToModel(&recordDTO)
		m[recordDTO.OrderID] = append(m[recordDTO.OrderID], *orderRecordModel)
	}

	var orders []*pvz_model.Order

	for _, dto := range orderDTOs {
		orderModel := transformOrderDtoToModel(&dto)
		orderModel.History = m[orderModel.ID]
		orders = append(orders, orderModel)
	}

	return orders, nil
}

func (r *OrderRepo) GetByIDs(ctx context.Context, ids []int64) ([]*pvz_model.Order, error) {
	if len(ids) == 0 {
		return []*pvz_model.Order{}, nil
	}

	var orderDTOs []orderDTO
	err := r.db.Select(ctx, &orderDTOs, `SELECT * FROM orders WHERE id = ANY($1) ORDER BY id ASC`, ids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*pvz_model.Order{}, nil
		}
		log.Print(err)
		return nil, err
	}

	var orders []*pvz_model.Order

	for _, dto := range orderDTOs {
		orders = append(orders, transformOrderDtoToModel(&dto))
	}

	return orders, nil
}

func (r *OrderRepo) GetByID(ctx context.Context, id int64) (*pvz_model.Order, error) {
	var a orderDTO
	err := r.db.Get(ctx, &a, "SELECT * FROM orders WHERE id=$1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		log.Print(err)
		return nil, err
	}
	return transformOrderDtoToModel(&a), nil
}

func (r *OrderRepo) SeedOrders(ctx context.Context) {
	now := time.Now()

	history := []pvz_model.OrderRecord{
		{
			Timestamp:   now.Add(-2 * time.Hour),
			Status:      pvz_model.OrderStatusReceived,
			Description: "Получено от курьера",
		},
	}

	for i := 0; i < 10; i++ {
		var orderID int64

		err := r.db.ExecQueryRow(ctx, `
			INSERT INTO orders
			(recipient_id, expiration_date, delivered_date, refunded_date, returned_date, status, weight, worth)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id`,
			58,                             // recipient
			now.Add(48*time.Hour),          // expiration
			now.Add(-1*time.Hour),          // delivered_date
			nil,                            // refunded_date
			nil,                            // returned_date
			pvz_model.OrderStatusDelivered, // status
			1.5,                            // weight
			2500.0,                         // worth
		).Scan(&orderID)

		if err != nil {
			log.Fatal("insert order: %w", err)
		}

		for _, rec := range history {
			_, err := r.db.Exec(ctx, `
				INSERT INTO order_records (order_id, timestamp, status, description)
				VALUES ($1, $2, $3, $4)`,
				orderID,
				rec.Timestamp,
				rec.Status,
				rec.Description,
			)

			if err != nil {
				log.Fatal("insert record: %w", err)
			}
		}
	}

	log.Println("Orders seeded successfully")
}
