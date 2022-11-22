package wallet

import (
	"context"

	"github.com/paper-trade-chatbot/be-match/config"
	"github.com/paper-trade-chatbot/be-proto/wallet"
)

type WalletIntf interface {
	CreateWallet(ctx context.Context, in *wallet.CreateWalletReq) (*wallet.CreateWalletRes, error)
	GetWallets(ctx context.Context, in *wallet.GetWalletsReq) (*wallet.GetWalletsRes, error)
	DeleteWallet(ctx context.Context, in *wallet.DeleteWalletReq) (*wallet.DeleteWalletRes, error)
	Transaction(ctx context.Context, in *wallet.TransactionReq) (*wallet.TransactionRes, error)
	RollbackTransaction(ctx context.Context, in *wallet.RollbackTransactionReq) (*wallet.RollbackTransactionRes, error)
	GetTransactionRecord(ctx context.Context, in *wallet.GetTransactionRecordReq) (*wallet.GetTransactionRecordRes, error)
	GetTransactionRecords(ctx context.Context, in *wallet.GetTransactionRecordsReq) (*wallet.GetTransactionRecordsRes, error)
}

type WalletImpl struct {
	WalletClient wallet.WalletServiceClient
}

var (
	WalletServiceHost    = config.GetString("WALLET_GRPC_HOST")
	WalletServerGRpcPort = config.GetString("WALLET_GRPC_PORT")
)

func New(walletClient wallet.WalletServiceClient) WalletIntf {
	return &WalletImpl{
		WalletClient: walletClient,
	}
}

func (impl *WalletImpl) CreateWallet(ctx context.Context, in *wallet.CreateWalletReq) (*wallet.CreateWalletRes, error) {
	return impl.WalletClient.CreateWallet(ctx, in)
}

func (impl *WalletImpl) GetWallets(ctx context.Context, in *wallet.GetWalletsReq) (*wallet.GetWalletsRes, error) {
	return impl.WalletClient.GetWallets(ctx, in)
}

func (impl *WalletImpl) DeleteWallet(ctx context.Context, in *wallet.DeleteWalletReq) (*wallet.DeleteWalletRes, error) {
	return impl.WalletClient.DeleteWallet(ctx, in)
}

func (impl *WalletImpl) Transaction(ctx context.Context, in *wallet.TransactionReq) (*wallet.TransactionRes, error) {
	return impl.WalletClient.Transaction(ctx, in)
}

func (impl *WalletImpl) RollbackTransaction(ctx context.Context, in *wallet.RollbackTransactionReq) (*wallet.RollbackTransactionRes, error) {
	return impl.WalletClient.RollbackTransaction(ctx, in)
}

func (impl *WalletImpl) GetTransactionRecord(ctx context.Context, in *wallet.GetTransactionRecordReq) (*wallet.GetTransactionRecordRes, error) {
	return impl.WalletClient.GetTransactionRecord(ctx, in)
}

func (impl *WalletImpl) GetTransactionRecords(ctx context.Context, in *wallet.GetTransactionRecordsReq) (*wallet.GetTransactionRecordsRes, error) {
	return impl.WalletClient.GetTransactionRecords(ctx, in)
}
