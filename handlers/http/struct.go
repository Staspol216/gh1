package http

import (
	"errors"
	"net/http"

	"github.com/Staspol216/gh1/models/order"
	"github.com/go-chi/render"
)

// Error

type ErrResponse struct {
	Err            error  `json:"-"`               // low-level runtime error
	HTTPStatusCode int    `json:"-"`               // http response status code
	StatusText     string `json:"status"`          // user-level status message
	AppCode        int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText      string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Internal error",
		ErrorText:      err.Error(),
	}
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}

// Order Response

type OrderResponse struct {
	*order.Order

	Elapsed int64 `json:"elapsed"`
}

func (rd *OrderResponse) Render(w http.ResponseWriter, r *http.Request) error {
	rd.Elapsed = 10
	return nil
}

func NewOrdersListResponse(orders []*order.Order) []render.Renderer {
	list := []render.Renderer{}
	for _, order := range orders {
		list = append(list, NewOrderResponse(order))
	}
	return list
}

func NewOrderResponse(o *order.Order) *OrderResponse {
	response := &OrderResponse{Order: o}
	return response
}

// OrderCreateRequest

type OrderCreateRequest struct {
	Order            *order.OrderParams `json:"order"`
	PackagingType    string             `json:"packagingType"`
	MembranaIncluded bool               `json:"membranaIncluded"`
}

func (a *OrderCreateRequest) Bind(r *http.Request) error {
	if a.Order == nil {
		return errors.New("missing required Order fields")
	}

	return nil
}

// OrderCreateRequest

type OrderUpdateRequest struct {
	OrderIDs    []int64 `json:"order_ids"`
	RecipientID int64   `json:"recipient_id"`
	Action      string  `json:"action"`
}

func (a *OrderUpdateRequest) Bind(r *http.Request) error {
	return nil
}

type OrderUpdateResponse struct{}

func NewOrderUpdateResponse() *OrderUpdateResponse {
	return &OrderUpdateResponse{}
}

func (rd *OrderUpdateResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)
	return nil
}

// OrderDeletedRequest

type OrderDeletedResponse struct{}

func (rd *OrderDeletedResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)
	return nil
}

func NewOrderDeletedResponse() *OrderDeletedResponse {
	return &OrderDeletedResponse{}
}

// OrderIDResponse

type OrderIDResponse struct {
	OrderID int64 `json:"order_id"`
}

func (rd *OrderIDResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusCreated)
	return nil
}

func NewOrderIDResponse(id int64) *OrderIDResponse {
	return &OrderIDResponse{OrderID: id}
}
