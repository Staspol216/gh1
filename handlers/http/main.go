package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	Serivces "github.com/Staspol216/gh1/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HTTPHandler struct {
	pvz *Serivces.Pvz
}

func New(p *Serivces.Pvz) *HTTPHandler {
	return &HTTPHandler{pvz: p}
}

func (h *HTTPHandler) Serve() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	r.Route("/orders", func(r chi.Router) {
		r.Get("/", h.ListOrders)

		r.Post("/", h.CreateOrder)

		r.Patch("/", h.UpdateOrders)

		r.Route("/{orderID}", func(r chi.Router) {
			r.Use(OrderCtx)

			r.Get("/", h.GetOrderByID)

			r.Delete("/", h.DeleteOrder)
		})

		r.Route("/refunds", func(r chi.Router) {
			r.Get("/", h.ListRefundedOrders)
		})
	})

	r.Route("/orders-history", func(r chi.Router) {
		r.Get("/", h.ListOrders)
	})

	host := os.Getenv("BACKEND_HOST")
	port := os.Getenv("BACKEND_PORT")

	err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), r)

	if err != nil {
		fmt.Println(err)
	}
}

type ctxKey string

const (
	ctxKeyOrderID ctxKey = "orderID"
)

func OrderCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var orderID int64

		id := chi.URLParam(r, "orderID")
		if id == "" {
			render.Render(w, r, ErrNotFound)
			return
		}

		parsedId, parseIntErr := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if parseIntErr != nil {
			render.Render(w, r, ErrInternal(parseIntErr))
			return
		}

		orderID = parsedId

		ctx := context.WithValue(r.Context(), ctxKeyOrderID, orderID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *HTTPHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders := h.pvz.GetOrders()

	err := render.RenderList(w, r, NewOrdersListResponse(orders))
	if err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (h *HTTPHandler) ListOrdersHistory(w http.ResponseWriter, r *http.Request) {
	orders := h.pvz.GetHistory()

	err := render.RenderList(w, r, NewOrdersListResponse(orders))
	if err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (h *HTTPHandler) ListRefundedOrders(w http.ResponseWriter, r *http.Request) {
	orders := h.pvz.GetAllRefunds()

	err := render.RenderList(w, r, NewOrdersListResponse(orders))
	if err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (h *HTTPHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	orderID, ok := r.Context().Value(ctxKeyOrderID).(int64)
	if !ok {
		render.Render(w, r, ErrInternal(errors.New("cannot get order id from requset context")))
		return
	}

	order, err := h.pvz.GetOrderByID(orderID)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	renderErr := render.Render(w, r, NewOrderResponse(order))
	if renderErr != nil {
		render.Render(w, r, ErrRender(renderErr))
	}
}

func (h *HTTPHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	data := &OrderCreateRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	orderId := h.pvz.AcceptFromCourier(data.Order, data.PackagingType, data.MembranaIncluded)

	err := render.Render(w, r, NewOrderIDResponse(orderId))
	if err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (h *HTTPHandler) UpdateOrders(w http.ResponseWriter, r *http.Request) {
	data := &OrderUpdateRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	err := h.pvz.ServeRecipient(data.OrderIDs, data.RecipientID, data.Action)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
	}

	renderErr := render.Render(w, r, NewOrderUpdateResponse())
	if renderErr != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (h *HTTPHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := r.Context().Value(ctxKeyOrderID).(int64)
	if !ok {
		render.Render(w, r, ErrInternal(errors.New("cannot get order id from requset context")))
		return
	}

	returnErr := h.pvz.ReturnToCourier(orderID)
	if returnErr != nil {
		render.Render(w, r, ErrInternal(returnErr))
	}

	renderErr := render.Render(w, r, NewOrderDeletedResponse())
	if renderErr != nil {
		render.Render(w, r, ErrRender(renderErr))
	}
}
