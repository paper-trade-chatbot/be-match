package order

import (
	"context"

	"github.com/paper-trade-chatbot/be-match/config"
	"github.com/paper-trade-chatbot/be-proto/order"
)

type OrderIntf interface {
	StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error)
	FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error)
	StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error)
	FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error)
	FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error)
	RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error)
	GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error)
}

type OrderImpl struct {
	OrderClient order.OrderServiceClient
}

var (
	OrderServiceHost    = config.GetString("ORDER_GRPC_HOST")
	OrderServerGRpcPort = config.GetString("ORDER_GRPC_PORT")
)

func New(orderClient order.OrderServiceClient) OrderIntf {
	return &OrderImpl{
		OrderClient: orderClient,
	}
}

func (impl *OrderImpl) StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error) {
	return impl.OrderClient.StartOpenPositionOrder(ctx, in)
}

func (impl *OrderImpl) FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error) {
	return impl.OrderClient.FinishOpenPositionOrder(ctx, in)
}

func (impl *OrderImpl) StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error) {
	return impl.OrderClient.StartClosePositionOrder(ctx, in)
}

func (impl *OrderImpl) FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error) {
	return impl.OrderClient.FinishClosePositionOrder(ctx, in)
}

func (impl *OrderImpl) FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error) {
	return impl.OrderClient.FailOrder(ctx, in)
}

func (impl *OrderImpl) RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error) {
	return impl.OrderClient.RollbackOrder(ctx, in)
}

func (impl *OrderImpl) GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error) {
	return impl.OrderClient.GetOrders(ctx, in)
}
