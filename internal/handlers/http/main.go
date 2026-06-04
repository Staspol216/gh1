package pvz_http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HTTPHandler struct {
	pvz     *pvz_order_service.PvzService
	context context.Context
}

func New(context context.Context, p *pvz_order_service.PvzService) *HTTPHandler {
	return &HTTPHandler{pvz: p, context: context}
}

func (h *HTTPHandler) Serve(cfg *pvz_config.Config) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("pong"))
		if err != nil {
			return
		}
	})

	r.Route("/orders", func(r chi.Router) {
		r.With(paginate).Get("/", h.ListOrders)

		r.With(logger).Post("/", h.CreateOrder)

		r.With(logger).Patch("/", h.UpdateOrders)

		r.Route("/{orderID}", func(r chi.Router) {
			r.Use(OrderCtx)

			r.With(logger).Get("/", h.GetOrder)

			r.With(logger).Delete("/", h.DeleteOrder)
		})

		r.Route("/refunds", func(r chi.Router) {
			r.Get("/", h.ListRefundedOrders)
		})
	})

	r.Route("/orders-history", func(r chi.Router) {
		r.Get("/", h.ListOrders)
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: r,
	}

	log.Println("HTTP server starting on", cfg.HTTPAddr())

	// Start server in goroutine so we can listen for shutdown signal
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-h.context.Done()

	log.Println("Shutdown signal received, gracefully shutting down HTTP server...")

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	return nil
}

type ctxKey string

const (
	ctxKeyOrderID ctxKey = "orderID"
	ctxKeyOffset  ctxKey = "offset"
	ctxKeyLimit   ctxKey = "limit"
)

const recipientIDQueryKey = "recipientID"

func OrderCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "orderID")
		if id == "" {
			err := render.Render(w, r, ErrInvalidRequest(errors.New("order id is required")))
			if err != nil {
				return
			}
			return
		}

		parsedOrderId, parseIntErr := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if parseIntErr != nil {
			err := render.Render(w, r, ErrInternal(parseIntErr))
			if err != nil {
				return
			}
			return
		}

		recipientID := r.URL.Query().Get(recipientIDQueryKey)
		if recipientID == "" {
			err := render.Render(w, r, ErrInvalidRequest(errors.New("user id is required")))
			if err != nil {
				return
			}
			return
		}

		parsedRecipientId, parseIntErr := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if parseIntErr != nil {
			err := render.Render(w, r, ErrInternal(parseIntErr))
			if err != nil {
				return
			}
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyOrderID, parsedOrderId)
		ctx = context.WithValue(ctx, recipientIDQueryKey, parsedRecipientId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)

		defer func(Body io.ReadCloser) {
			cErr := Body.Close()
			if cErr != nil {
				return
			}
		}(r.Body)

		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		log.Printf("Remote: %s %s", r.RemoteAddr, r.UserAgent())
		log.Printf("Request: %s %s %s", r.Method, r.URL.Path, r.Proto)

		if len(body) > 0 {
			log.Printf("Body: %s", string(body))
		}

		next.ServeHTTP(w, r)
	})
}

func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const (
			defaultOffset int64 = 0
			defaultLimit  int64 = 10
			maxLimit      int64 = 100
		)

		q := r.URL.Query()
		offset := defaultOffset
		limit := defaultLimit

		if os := strings.TrimSpace(q.Get("offset")); os != "" {
			if v, err := strconv.ParseInt(os, 10, 64); err == nil && v >= 0 {
				offset = v
			}
		}

		if ls := strings.TrimSpace(q.Get("limit")); ls != "" {
			if v, err := strconv.ParseInt(ls, 10, 64); err == nil && v > 0 {
				limit = min(v, maxLimit)
			}
		}

		ctx := context.WithValue(r.Context(), ctxKeyOffset, offset)
		ctx = context.WithValue(ctx, ctxKeyLimit, limit)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *HTTPHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	offset, _ := r.Context().Value(ctxKeyOffset).(int64)
	limit, _ := r.Context().Value(ctxKeyLimit).(int64)

	pagination := &pvz_domain.Pagination{
		Offset: offset,
		Limit:  limit,
	}

	orders, getOrdersErr := h.pvz.GetOrders(r.Context(), pagination)

	if getOrdersErr != nil {
		eErr := render.Render(w, r, ErrInternal(getOrdersErr))
		if eErr != nil {
			return
		}
		return
	}

	err := render.RenderList(w, r, NewOrdersListResponse(orders))
	if err != nil {
		rErr := render.Render(w, r, ErrRender(err))
		if rErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) ListOrdersHistory(w http.ResponseWriter, r *http.Request) {
	offset, _ := r.Context().Value(ctxKeyOffset).(int64)
	limit, _ := r.Context().Value(ctxKeyLimit).(int64)

	pagination := &pvz_domain.Pagination{
		Offset: offset,
		Limit:  limit,
	}

	orders, err := h.pvz.GetHistory(r.Context(), pagination)

	if err != nil {
		eErr := render.Render(w, r, ErrInternal(err))
		if eErr != nil {
			return
		}
		return
	}

	renderErr := render.RenderList(w, r, NewOrdersListResponse(orders))
	if renderErr != nil {
		eErr := render.Render(w, r, ErrRender(renderErr))
		if eErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) ListRefundedOrders(w http.ResponseWriter, r *http.Request) {
	offset, _ := r.Context().Value(ctxKeyOffset).(int64)
	limit, _ := r.Context().Value(ctxKeyLimit).(int64)

	pagination := &pvz_domain.Pagination{
		Offset: offset,
		Limit:  limit,
	}

	orders, err := h.pvz.GetAllRefunds(r.Context(), pagination)

	if err != nil {
		eErr := render.Render(w, r, ErrInternal(err))
		if eErr != nil {
			return
		}
		return
	}

	renderErr := render.RenderList(w, r, NewOrdersListResponse(orders))
	if renderErr != nil {
		eErr := render.Render(w, r, ErrRender(renderErr))
		if eErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := r.Context().Value(ctxKeyOrderID).(int64)
	if !ok {
		err := render.Render(w, r, ErrInternal(errors.New("cannot get order id from request context")))
		if err != nil {
			return
		}
		return
	}

	recipientID, ok := r.Context().Value(recipientIDQueryKey).(int64)
	if !ok {
		eErr := render.Render(w, r, ErrInternal(errors.New("cannot get recipient id from request context")))
		if eErr != nil {
			return
		}
		return
	}

	order, err := h.pvz.GetOrderByID(r.Context(), orderID, recipientID)
	if err != nil {
		eErr := render.Render(w, r, ErrInternal(err))
		if eErr != nil {
			return
		}
		return
	}

	renderErr := render.Render(w, r, NewOrderResponse(order))
	if renderErr != nil {
		eErr := render.Render(w, r, ErrRender(renderErr))
		if eErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	data := &OrderCreateRequest{}
	if err := render.Bind(r, data); err != nil {
		eErr := render.Render(w, r, ErrInvalidRequest(err))
		if eErr != nil {
			return
		}
		return
	}

	if res := time.Now().Compare(data.Order.ExpirationDate); res == 1 {
		err := render.Render(w, r, ErrInvalidRequest(errors.New("expiration date can't be in the past")))
		if err != nil {
			return
		}
		return
	}

	orderId, err := h.pvz.AcceptFromCourier(r.Context(), data.Order, data.PackagingType, data.MembranaIncluded)

	if err != nil {
		if eErr := render.Render(w, r, ErrInternal(err)); eErr != nil {
			return
		}
	}

	renderError := render.Render(w, r, NewOrderIDResponse(*orderId))
	if renderError != nil {
		if eErr := render.Render(w, r, ErrRender(renderError)); eErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) UpdateOrders(w http.ResponseWriter, r *http.Request) {
	data := &OrderUpdateRequest{}
	if err := render.Bind(r, data); err != nil {
		if eErr := render.Render(w, r, ErrInvalidRequest(err)); eErr != nil {
			return
		}
	}

	err := h.pvz.ServeRecipient(r.Context(), data.OrderIDs, data.RecipientID, data.Action)
	if err != nil {
		if rErr := render.Render(w, r, ErrInternal(err)); rErr != nil {
			return
		}
	}

	renderErr := render.Render(w, r, NewOrderUpdateResponse())
	if renderErr != nil {
		rErr := render.Render(w, r, ErrRender(renderErr))
		if rErr != nil {
			return
		}
	}
}

func (h *HTTPHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := r.Context().Value(ctxKeyOrderID).(int64)
	if !ok {
		err := render.Render(w, r, ErrInternal(errors.New("cannot get order id from request context")))
		if err != nil {
			return
		}
		return
	}

	returnErr := h.pvz.ReturnToCourier(r.Context(), orderID)
	if returnErr != nil {
		err := render.Render(w, r, ErrInternal(returnErr))
		if err != nil {
			return
		}
	}

	renderErr := render.Render(w, r, NewOrderDeletedResponse())
	if renderErr != nil {
		err := render.Render(w, r, ErrRender(renderErr))
		if err != nil {
			return
		}
	}
}
