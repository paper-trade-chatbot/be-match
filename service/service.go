package service

import (
	"context"
	"fmt"

	"github.com/paper-trade-chatbot/be-common/logging"

	"github.com/paper-trade-chatbot/be-common/config"
	"github.com/paper-trade-chatbot/be-match/service/member"
	"github.com/paper-trade-chatbot/be-match/service/order"
	"github.com/paper-trade-chatbot/be-match/service/position"
	"github.com/paper-trade-chatbot/be-match/service/product"
	"github.com/paper-trade-chatbot/be-match/service/quote"
	"github.com/paper-trade-chatbot/be-match/service/wallet"
	memberGrpc "github.com/paper-trade-chatbot/be-proto/member"
	orderGrpc "github.com/paper-trade-chatbot/be-proto/order"
	positionGrpc "github.com/paper-trade-chatbot/be-proto/position"
	productGrpc "github.com/paper-trade-chatbot/be-proto/product"
	quoteGrpc "github.com/paper-trade-chatbot/be-proto/quote"
	walletGrpc "github.com/paper-trade-chatbot/be-proto/wallet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var Impl ServiceImpl
var (
	MemberServiceHost    = config.GetString("MEMBER_GRPC_HOST")
	MemberServerGRpcPort = config.GetString("MEMBER_GRPC_PORT")
	memberServiceConn    *grpc.ClientConn

	ProductServiceHost    = config.GetString("PRODUCT_GRPC_HOST")
	ProductServerGRpcPort = config.GetString("PRODUCT_GRPC_PORT")
	productServiceConn    *grpc.ClientConn

	OrderServiceHost    = config.GetString("ORDER_GRPC_HOST")
	OrderServerGRpcPort = config.GetString("ORDER_GRPC_PORT")
	orderServiceConn    *grpc.ClientConn

	WalletServiceHost    = config.GetString("WALLET_GRPC_HOST")
	WalletServerGRpcPort = config.GetString("WALLET_GRPC_PORT")
	walletServiceConn    *grpc.ClientConn

	QuoteServiceHost    = config.GetString("QUOTE_GRPC_HOST")
	QuoteServerGRpcPort = config.GetString("QUOTE_GRPC_PORT")
	quoteServiceConn    *grpc.ClientConn

	PositionServiceHost    = config.GetString("POSITION_GRPC_HOST")
	PositionServerGRpcPort = config.GetString("POSITION_GRPC_PORT")
	positionServiceConn    *grpc.ClientConn
)

type ServiceImpl struct {
	MemberIntf   member.MemberIntf
	ProductIntf  product.ProductIntf
	OrderIntf    order.OrderIntf
	WalletIntf   wallet.WalletIntf
	QuoteIntf    quote.QuoteIntf
	PositionIntf position.PositionIntf
}

func GrpcDial(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(20*1024*1024),
		grpc.MaxCallSendMsgSize(20*1024*1024)), grpc.WithUnaryInterceptor(clientInterceptor))
}

func Initialize(ctx context.Context) {

	var err error

	addr := MemberServiceHost + ":" + MemberServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	memberServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	memberConn := memberGrpc.NewMemberServiceClient(memberServiceConn)
	Impl.MemberIntf = member.New(memberConn)

	addr = ProductServiceHost + ":" + ProductServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	productServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	productConn := productGrpc.NewProductServiceClient(productServiceConn)
	Impl.ProductIntf = product.New(productConn)

	addr = OrderServiceHost + ":" + OrderServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	orderServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	orderConn := orderGrpc.NewOrderServiceClient(orderServiceConn)
	Impl.OrderIntf = order.New(orderConn)

	addr = WalletServiceHost + ":" + WalletServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	walletServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	walletConn := walletGrpc.NewWalletServiceClient(walletServiceConn)
	Impl.WalletIntf = wallet.New(walletConn)

	addr = QuoteServiceHost + ":" + QuoteServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	quoteServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	quoteConn := quoteGrpc.NewQuoteServiceClient(quoteServiceConn)
	Impl.QuoteIntf = quote.New(quoteConn)

	addr = PositionServiceHost + ":" + PositionServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	positionServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	positionConn := positionGrpc.NewPositionServiceClient(positionServiceConn)
	Impl.PositionIntf = position.New(positionConn)
}

func Finalize(ctx context.Context) {
	memberServiceConn.Close()
	productServiceConn.Close()
	orderServiceConn.Close()
	walletServiceConn.Close()
	quoteServiceConn.Close()
}

func clientInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	requestId, _ := ctx.Value(logging.ContextKeyRequestId).(string)
	account, _ := ctx.Value(logging.ContextKeyAccount).(string)

	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		logging.ContextKeyRequestId: requestId,
		logging.ContextKeyAccount:   account,
	}))

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		fmt.Println("clientInterceptor err:", err.Error())
	}

	return err
}
