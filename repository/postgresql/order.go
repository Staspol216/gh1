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
	query := `INSERT INTO public.orders (
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

func (r *OrderRepo) Delete(orderId int64) error {
	commandTag, err := r.db.Exec(r.context, `--sql DELETE FROM orders WHERE ID = $1;`, orderId)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("no row found to delete")
	}
	return nil
}

func (r *OrderRepo) Update(updatedOrder *order.Order) {

}

func (r *OrderRepo) GetList() []*order.Order {
	var orderDTOs []orderDTO
	err := r.db.Get(r.context, &orderDTOs, `--sql SELECT * FROM orders`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		log.Print(err)
		return nil
	}

	var orders []*order.Order

	for _, dto := range orderDTOs {
		orders = append(orders, transformDtoToModel(&dto))
	}

	return orders
}

func (r *OrderRepo) GetByRecipientAndIds(recipientId int64, ids []int64) ([]*order.Order, error) {
	var orderDTOs []orderDTO
	err := r.db.Get(r.context, &orderDTOs, `--sql SELECT * FROM orders WHERE recipient_id = $1 AND id = ANY($2)`, recipientId, ids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		log.Print(err)
		return nil, err
	}

	var orders []*order.Order

	for _, dto := range orderDTOs {
		orders = append(orders, transformDtoToModel(&dto))
	}

	return orders, nil
}

func (r *OrderRepo) GetByID(id int64) (*order.Order, error) {
	var a orderDTO
	err := r.db.Get(r.context, &a, `--sql SELECT * FROM orders WHERE id=$1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		log.Print(err)
		return nil, err
	}
	return transformDtoToModel(&a), nil
}
