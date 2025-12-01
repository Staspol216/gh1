package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/Staspol216/gh1/db"
	"github.com/Staspol216/gh1/models/order"
)

type OrderRepo struct {
	db      db.DB
	context context.Context
}

func NewOrderRepo(database db.DB, context context.Context) (*OrderRepo, error) {
	return &OrderRepo{
		db:      database,
		context: context,
	}, nil
}

func (r *OrderRepo) Add(order *order.Order) (int64, error) {
	query := `INSERT INTO orders (
		recipient_id,
		expiration_date,
		status,
		weight,
		worth
	) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	row := r.db.ExecQueryRow(r.context, query,
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

func (r *OrderRepo) AddHistoryRecord(record *order.OrderRecord, orderId int64) (int64, error) {
	query := `INSERT INTO order_records (
		order_id,
		description,
		timestamp,
		status
	) VALUES ($1, $2, $3, $4) RETURNING id;`

	row := r.db.ExecQueryRow(r.context, query,
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

func (r *OrderRepo) Delete(orderId int64) error {
	commandTag, err := r.db.Exec(r.context, `DELETE FROM orders WHERE ID = $1;`, orderId)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("no row found to delete")
	}
	return nil
}

func (r *OrderRepo) Update(updatedOrder *order.Order) error {
	query := `UPDATE orders SET recipient_id=$1, expiration_date=$2, delivered_date=$3, refunded_date=$4, returned_date=$5, status=$6, weight=$7, worth=$8 WHERE id=$9 RETURNING id;`

	row := r.db.ExecQueryRow(r.context, query,
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

func (r *OrderRepo) GetList() []*order.Order {

	var orderDTOs []orderDTO
	err := r.db.Select(r.context, &orderDTOs, `
		SELECT * 
		FROM orders
	`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		log.Print(err)
		return nil
	}

	var recordDTOs []orderRecordDTO
	orderRecordsErr := r.db.Select(r.context, &recordDTOs, `
        SELECT id, order_id, timestamp, status, description
        FROM order_records
        ORDER BY order_id, timestamp
    `)
	if orderRecordsErr != nil {
		log.Println(orderRecordsErr)
		return nil
	}

	m := make(map[int64][]order.OrderRecord)
	for _, recordDTO := range recordDTOs {
		orderRecordModel := transformOrderRecordDtoToModel(&recordDTO)
		m[recordDTO.OrderID] = append(m[recordDTO.OrderID], *orderRecordModel)
	}

	var orders []*order.Order

	for _, dto := range orderDTOs {
		orderModel := transformOrderDtoToModel(&dto)
		orderModel.History = m[orderModel.ID]
		orders = append(orders, orderModel)
	}

	return orders
}

func (r *OrderRepo) GetByRecipientId(recipientId int64) ([]*order.Order, error) {
	var orderDTOs []orderDTO
	err := r.db.Select(r.context, &orderDTOs, `SELECT * FROM orders WHERE recipient_id = $1`, recipientId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		log.Print(err)
		return nil, err
	}

	var orders []*order.Order

	for _, dto := range orderDTOs {
		orders = append(orders, transformOrderDtoToModel(&dto))
	}

	return orders, nil
}

func (r *OrderRepo) GetByID(id int64) (*order.Order, error) {
	var a orderDTO
	err := r.db.Get(r.context, &a, `SELECT * FROM orders WHERE id=$1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		log.Print(err)
		return nil, err
	}
	return transformOrderDtoToModel(&a), nil
}
