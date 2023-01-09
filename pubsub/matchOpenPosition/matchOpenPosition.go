package matchOpenPosition

import (
	"context"
	"database/sql"
	"time"

	common "github.com/paper-trade-chatbot/be-common"
	"github.com/paper-trade-chatbot/be-common/database"
	"github.com/paper-trade-chatbot/be-common/logging"
	"github.com/paper-trade-chatbot/be-match/dao/matchRecordDao"
	"github.com/paper-trade-chatbot/be-match/models/dbModels"
	"github.com/paper-trade-chatbot/be-match/service"
	"github.com/paper-trade-chatbot/be-proto/order"
	"github.com/paper-trade-chatbot/be-proto/product"
	"github.com/paper-trade-chatbot/be-proto/quote"
	"github.com/paper-trade-chatbot/be-proto/wallet"
	"github.com/paper-trade-chatbot/be-pubsub/order/openPosition/rabbitmq"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/status"
)

func MatchOpenPosition(ctx context.Context, model *rabbitmq.OpenPositionModel) error {
	logging.Info(ctx, "[MatchOpenPosition] model: %#v", model)
	db := database.GetDB()
	deal := false
	retryCount := 0
	var transactionID uint64 = 0
	unitPrice := decimal.Decimal{}
	var orderErr error
	orderProcess := order.OrderProcess_OrderProcess_Failed
	var expire *int64

	if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
		Id:           model.ID,
		OrderProcess: order.OrderProcess_OrderProcess_Matching,
	}); err != nil {
		logging.Error(ctx, "[MatchOpenPosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
	}

	defer func() {
		if orderErr != nil {
			failCode := uint64(status.Code(orderErr))
			s, _ := status.FromError(orderErr)
			remark := s.Message()
			if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
				Id:       model.ID,
				FailCode: &failCode,
				Remark:   &remark,
			}); err != nil {
				logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
			}
			if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
				Id:           model.ID,
				OrderProcess: order.OrderProcess_OrderProcess_Failed,
			}); err != nil {
				logging.Error(ctx, "[MatchOpenPosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
			}
			return
		}

		if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
			Id:           model.ID,
			OrderProcess: orderProcess,
			Expire:       expire,
		}); err != nil {
			logging.Error(ctx, "[MatchOpenPosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
		}
	}()

	matchRecord := &dbModels.MatchRecordModel{
		OrderID:         model.ID,
		MemberID:        model.MemberID,
		MatchStatus:     dbModels.MatchStatus_Pending,
		TransactionType: dbModels.TransactionType_OpenPosition,
		ExchangeCode:    model.ExchangeCode,
		ProductCode:     model.ProductCode,
		TradeType:       dbModels.TradeType(model.TradeType),
		Amount:          model.Amount,
	}

	if _, err := matchRecordDao.New(db, matchRecord); err != nil {
		logging.Error(ctx, "[MatchOpenPosition] failed to new matchRecord: %v", err)
		orderErr = err
		return err
	}
	var openPrice *decimal.NullDecimal = nil
	var positionID *sql.NullInt64 = nil

	defer func() {
		if matchRecord.MatchStatus != dbModels.MatchStatus_Finished {
			matchRecord.MatchStatus = dbModels.MatchStatus_Failed
		}

		if err := matchRecordDao.Modify(db, matchRecord, &matchRecordDao.UpdateModel{
			MatchStatus: &matchRecord.MatchStatus,
			PositionID:  positionID,
			OpenPrice:   openPrice,
		}); err != nil {
			logging.Error(ctx, "[MatchOpenPosition] failed to Modify matchRecord [%d]: %v", model.ID, err)
		}
	}()

	productRes, err := service.Impl.ProductIntf.GetProduct(ctx, &product.GetProductReq{
		Product: &product.GetProductReq_Code{
			Code: &product.ExchangeCodeProductCode{
				ExchangeCode: model.ExchangeCode,
				ProductCode:  model.ProductCode,
			},
		},
	})

	if err != nil {
		logging.Error(ctx, "[MatchOpenPosition] failed to get product [%s][%s]: %v", model.ExchangeCode, model.ProductCode, err)
		orderErr = err
		return err
	}

	for !deal && retryCount <= 10 {

		retryCount++

		walletRes, err := service.Impl.WalletIntf.GetWallets(ctx, &wallet.GetWalletsReq{
			Wallet: &wallet.GetWalletsReq_MemberID{
				MemberID: model.MemberID,
			},
			Currency: &productRes.Product.CurrencyCode,
		})
		if err != nil || len(walletRes.Wallets) == 0 {
			if err == nil {
				err = common.ErrNoSuchWallet
			}
			logging.Error(ctx, "[MatchOpenPosition] failed to get wallet by member[%d] currency[%s]: %v", model.MemberID, productRes.Product.CurrencyCode, err)
			orderErr = err
			return err
		}

		balance, err := decimal.NewFromString(walletRes.Wallets[0].Amount)
		if err != nil {
			logging.Error(ctx, "[MatchOpenPosition] NewFromString failed: %v", err)
			orderErr = err
			return err
		}

		flag := quote.GetQuotesReq_GetFlag_None
		key := ""
		if model.TradeType == rabbitmq.TradeType_Buy {
			flag = quote.GetQuotesReq_GetFlag_Ask
			key = "ask"
		} else {
			flag = quote.GetQuotesReq_GetFlag_Bid
			key = "bid"
		}

		getFrom := "000000"
		getTo := "000000"
		quoteRes, err := service.Impl.QuoteIntf.GetQuotes(ctx, &quote.GetQuotesReq{
			ProductIDs: []int64{productRes.Product.Id},
			Flag:       flag,
			GetFrom:    &getFrom,
			GetTo:      &getTo,
		})
		if err != nil || len(quoteRes.Quotes) == 0 {
			logging.Warn(ctx, "[MatchOpenPosition] GetQuotes failed. retry later: %v", err)
			continue
		}

		unitPriceString, ok := quoteRes.Quotes[0].Quotes[key]
		if !ok {
			logging.Warn(ctx, "[MatchOpenPosition] GetQuotes no [%s] field. retry later", key)
			continue
		}

		unitPrice, err = decimal.NewFromString(unitPriceString)
		if err != nil {
			logging.Warn(ctx, "[MatchOpenPosition] GetQuotes NewFromString failed. retry later: %v", err)
			continue
		}

		if balance.LessThan(unitPrice.Mul(model.Amount)) {
			logging.Error(ctx, "[MatchOpenPosition] balance not enough: %v", common.ErrInsufficientBalance)
			orderErr = common.ErrInsufficientBalance
			return err
		}

		beforeAmount := balance.String()
		transactionRes, err := service.Impl.WalletIntf.Transaction(ctx, &wallet.TransactionReq{
			WalletID:     walletRes.Wallets[0].Id,
			Action:       wallet.Action_Action_OPEN,
			Amount:       unitPrice.Mul(model.Amount).Neg().String(),
			Currency:     productRes.Product.CurrencyCode,
			CommitterID:  model.MemberID,
			BeforeAmount: &beforeAmount,
		})
		if err != nil {
			logging.Warn(ctx, "[MatchOpenPosition] Transaction failed. retry later: %v", err)
			continue
		}

		transactionID = transactionRes.Id
		deal = true
	}

	if !deal {
		logging.Error(ctx, "[MatchOpenPosition] failed to match [%d]: %v", model.ID, common.ErrExceedRetryTimes)
		orderErr = common.ErrExceedRetryTimes
		return common.ErrExceedRetryTimes
	}

	if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
		Id:           model.ID,
		OrderProcess: order.OrderProcess_OrderProcess_Matched,
	}); err != nil {
		logging.Error(ctx, "[MatchOpenPosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
	}

	res, err := service.Impl.OrderIntf.FinishOpenPositionOrder(ctx, &order.FinishOpenPositionOrderReq{
		Id:                  model.ID,
		UnitPrice:           unitPrice.String(),
		TransactionRecordID: uint64(transactionID),
		FinishedAt:          time.Now().Unix(),
	})
	if err != nil {
		logging.Error(ctx, "[MatchOpenPosition] FinishOpenPositionOrder failed: %v", err)
		if _, err := service.Impl.WalletIntf.RollbackTransaction(ctx, &wallet.RollbackTransactionReq{
			Id:           transactionID,
			RollbackerID: model.MemberID,
		}); err != nil {
			logging.Error(ctx, "[MatchOpenPosition] failed to RollbackTransaction [%d]: %v", model.ID, err)
		}
		return err
	}

	matchRecord.MatchStatus = dbModels.MatchStatus_Finished
	positionID = &sql.NullInt64{
		Valid: true,
		Int64: int64(res.PositionID),
	}
	openPrice = &decimal.NullDecimal{
		Valid:   true,
		Decimal: unitPrice,
	}

	orderProcess = order.OrderProcess_OrderProcess_Finished
	expireTime := int64(time.Minute)
	expire = &expireTime
	return nil
}
