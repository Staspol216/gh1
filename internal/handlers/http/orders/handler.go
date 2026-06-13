package orders_http_handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/Staspol216/gh1/pkg/api/openapi/common"
	"github.com/Staspol216/gh1/pkg/api/openapi/orders"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	oapi_middleware "github.com/oapi-codegen/nethttp-middleware"
	"go.uber.org/zap"
)

type Handler struct {
	pvz     *pvz_order_service.PvzService
	context context.Context
}

var _ orders_dto.StrictServerInterface = (*Handler)(nil)

func (h *Handler) GetOrders(ctx context.Context, request orders_dto.GetOrdersRequestObject) (orders_dto.GetOrdersResponseObject, error) {
	orders, getOrdersErr := h.pvz.GetOrders(ctx, paginationFromParams(request.Params.Offset, request.Params.Limit))

	if getOrdersErr != nil {
		return orders_dto.GetOrdersdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, getOrdersErr.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.GetOrders200JSONResponse{
		GetOrdersListResponseJSONResponse: ordersListResponse(orders),
	}, nil
}

func (h *Handler) UpdateOrders(ctx context.Context, request orders_dto.UpdateOrdersRequestObject) (orders_dto.UpdateOrdersResponseObject, error) {

	if err := h.pvz.ServeRecipient(ctx, request.Body.OrderIds, request.Body.RecipientId, string(request.Body.Action)); err != nil {
		return orders_dto.UpdateOrdersdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.UpdateOrders204Response{}, nil
}

func (h *Handler) CreateOrder(ctx context.Context, request orders_dto.CreateOrderRequestObject) (orders_dto.CreateOrderResponseObject, error) {

	orderParams := &pvz_domain.OrderParams{
		RecipientId:    request.Body.Order.RecipientId,
		ExpirationDate: request.Body.Order.ExpirationDate,
		Weight:         request.Body.Order.Weight,
		Worth:          request.Body.Order.Worth,
	}

	if time.Now().Compare(orderParams.ExpirationDate) == 1 {
		return orders_dto.CreateOrder400JSONResponse{
			ApiErrorResponseJSONResponse: common_dto.ApiErrorResponseJSONResponse(apiError(common_dto.BadRequest, "expiration date can't be in the past")),
		}, nil
	}

	orderID, err := h.pvz.AcceptFromCourier(ctx, orderParams, string(request.Body.PackagingType), request.Body.MembranaIncluded)
	if err != nil {
		return orders_dto.CreateOrderdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.CreateOrder201JSONResponse{
		CreateOrderResponseJSONResponse: orders_dto.CreateOrderResponseJSONResponse{
			OrderId: *orderID,
		},
	}, nil
}

func (h *Handler) GetOrdersHistory(ctx context.Context, request orders_dto.GetOrdersHistoryRequestObject) (orders_dto.GetOrdersHistoryResponseObject, error) {

	orders, err := h.pvz.GetHistory(ctx, paginationFromParams(request.Params.Offset, request.Params.Limit))
	if err != nil {
		return orders_dto.GetOrdersHistorydefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.GetOrdersHistory200JSONResponse{
		GetOrdersListResponseJSONResponse: ordersListResponse(orders),
	}, nil
}

func (h *Handler) GetRefundedOrders(ctx context.Context, request orders_dto.GetRefundedOrdersRequestObject) (orders_dto.GetRefundedOrdersResponseObject, error) {
	orders, err := h.pvz.GetAllRefunds(ctx, paginationFromParams(request.Params.Offset, request.Params.Limit))
	if err != nil {
		return orders_dto.GetRefundedOrdersdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.GetRefundedOrders200JSONResponse{
		GetOrdersListResponseJSONResponse: ordersListResponse(orders),
	}, nil
}

func (h *Handler) DeleteOrder(ctx context.Context, request orders_dto.DeleteOrderRequestObject) (orders_dto.DeleteOrderResponseObject, error) {
	if err := h.pvz.ReturnToCourier(ctx, request.OrderID); err != nil {
		return orders_dto.DeleteOrderdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.DeleteOrder204Response{}, nil
}

func (h *Handler) GetOrder(ctx context.Context, request orders_dto.GetOrderRequestObject) (orders_dto.GetOrderResponseObject, error) {
	order, err := h.pvz.GetOrderByID(ctx, request.OrderID, request.Params.RecipientID)
	if err != nil {
		return orders_dto.GetOrderdefaultJSONResponse{
			Body:       apiError(common_dto.Internal, err.Error()),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return orders_dto.GetOrder200JSONResponse{
		GetOrderResponseJSONResponse: orders_dto.GetOrderResponseJSONResponse(toOrderDTO(order)),
	}, nil
}

func (h *Handler) GetPing(_ context.Context, _ orders_dto.GetPingRequestObject) (orders_dto.GetPingResponseObject, error) {
	return orders_dto.GetPing200TextResponse("pong"), nil
}

func (h *Handler) Serve(cfg *pvz_config.Config) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(tracingMiddleware)
	r.Use(metricsMiddleware)

	swagger, err := orders_dto.GetSpec()
	if err != nil {
		app_logger.MyLogger.Fatal("Error loading swagger spec", zap.Error(err))
	}

	r.Use(oapi_middleware.OapiRequestValidator(swagger))

	srv := &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app_logger.MyLogger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-h.context.Done()

	app_logger.MyLogger.Info("shutdown signal received, gracefully shutting down HTTP server")

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		app_logger.MyLogger.Error("HTTP server shutdown error", zap.Error(err))
	}

	return nil
}

func NewHandler(service *pvz_order_service.PvzService, ctx context.Context) *Handler {
	return &Handler{
		pvz:     service,
		context: ctx,
	}
}

func paginationFromParams(offsetParam *orders_dto.Offset, limitParam *orders_dto.Limit) *pvz_domain.Pagination {
	const (
		defaultOffset int64 = 0
		defaultLimit  int64 = 10
		maxLimit      int64 = 100
	)

	offset := defaultOffset
	limit := defaultLimit

	if offsetParam != nil {
		if v, err := strconv.ParseInt(*offsetParam, 10, 64); err == nil && v >= 0 {
			offset = v
		}
	}

	if limitParam != nil {
		if v := int64(*limitParam); v > 0 {
			limit = min(v, maxLimit)
		}
	}

	return &pvz_domain.Pagination{
		Offset: offset,
		Limit:  limit,
	}
}

func ordersListResponse(orders []*pvz_domain.Order) orders_dto.GetOrdersListResponseJSONResponse {
	ordersDtos := make([]orders_dto.Order, 0, len(orders))
	for _, order := range orders {
		ordersDtos = append(ordersDtos, toOrderDTO(order))
	}

	return orders_dto.GetOrdersListResponseJSONResponse{
		Orders: ordersDtos,
	}
}

func apiError(code common_dto.ApiErrorCode, message string) common_dto.ApiError {
	return common_dto.ApiError{
		Error: common_dto.BaseError{
			Code:    code,
			Message: message,
		},
	}
}
