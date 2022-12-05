package matchClosePosition

import (
	"context"
	"database/sql"
	"time"

	common "github.com/paper-trade-chatbot/be-common"
	"github.com/paper-trade-chatbot/be-match/dao/matchRecordDao"
	"github.com/paper-trade-chatbot/be-match/database"
	"github.com/paper-trade-chatbot/be-match/logging"
	"github.com/paper-trade-chatbot/be-match/models/dbModels"
	"github.com/paper-trade-chatbot/be-match/service"
	"github.com/paper-trade-chatbot/be-proto/order"
	"github.com/paper-trade-chatbot/be-proto/position"
	"github.com/paper-trade-chatbot/be-proto/product"
	"github.com/paper-trade-chatbot/be-proto/quote"
	"github.com/paper-trade-chatbot/be-proto/wallet"
	"github.com/paper-trade-chatbot/be-pubsub/order/closePosition/rabbitmq"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/status"
)

func MatchClosePosition(ctx context.Context, model *rabbitmq.ClosePositionModel) error {
	logging.Info(ctx, "[MatchClosePosition] model: %#v", model)
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
		logging.Error(ctx, "[MatchClosePosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
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
			if _, err := service.Impl.PositionIntf.StopPendingPosition(ctx, &position.StopPendingPositionReq{
				Id: model.PositionID,
			}); err != nil {
				logging.Error(ctx, "[MatchClosePosition] StopPendingPosition failed: %v", err)
			}
			if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
				Id:           model.ID,
				OrderProcess: order.OrderProcess_OrderProcess_Failed,
			}); err != nil {
				logging.Error(ctx, "[MatchClosePosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
			}
			return
		}

		if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
			Id:           model.ID,
			OrderProcess: orderProcess,
			Expire:       expire,
		}); err != nil {
			logging.Error(ctx, "[MatchClosePosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
		}
	}()

	matchRecord := &dbModels.MatchRecordModel{
		OrderID:         model.ID,
		MemberID:        model.MemberID,
		PositionID:      sql.NullInt64{Valid: true, Int64: int64(model.PositionID)},
		MatchStatus:     dbModels.MatchStatus_Pending,
		TransactionType: dbModels.TransactionType_ClosePosition,
		ExchangeCode:    model.ExchangeCode,
		ProductCode:     model.ProductCode,
		TradeType:       dbModels.TradeType(model.TradeType),
		OpenPrice:       decimal.NewNullDecimal(model.OpenPrice),
		Amount:          model.CloseAmount,
	}

	var closePrice *decimal.NullDecimal = nil

	if _, err := matchRecordDao.New(db, matchRecord); err != nil {
		logging.Error(ctx, "[MatchClosePosition] failed to new matchRecord: %v", err)
	}

	defer func() {
		if matchRecord.MatchStatus != dbModels.MatchStatus_Finished {
			matchRecord.MatchStatus = dbModels.MatchStatus_Failed
		}

		if err := matchRecordDao.Modify(db, matchRecord, &matchRecordDao.UpdateModel{
			MatchStatus: &matchRecord.MatchStatus,
			ClosePrice:  closePrice,
		}); err != nil {
			logging.Error(ctx, "[MatchClosePosition] failed to Modify matchRecord [%d]: %v", model.ID, err)
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
		logging.Error(ctx, "[MatchClosePosition] failed to get product [%s][%s]: %v", model.ExchangeCode, model.ProductCode, err)
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
			logging.Error(ctx, "[MatchClosePosition] failed to get wallet by member[%d] currency[%s]: %v", model.MemberID, productRes.Product.CurrencyCode, err)
			orderErr = err
			return err
		}

		balance, err := decimal.NewFromString(walletRes.Wallets[0].Amount)
		if err != nil {
			logging.Error(ctx, "[MatchClosePosition] NewFromString failed: %v", err)
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
			logging.Warn(ctx, "[MatchClosePosition] GetQuotes failed. retry later: %v", err)
			continue
		}

		unitPriceString, ok := quoteRes.Quotes[0].Quotes[key]
		if !ok {
			logging.Warn(ctx, "[MatchClosePosition] GetQuotes no [%s] field. retry later", key)
			continue
		}

		unitPrice, err = decimal.NewFromString(unitPriceString)
		if err != nil {
			logging.Warn(ctx, "[MatchClosePosition] GetQuotes NewFromString failed. retry later: %v", err)
			continue
		}

		equity := decimal.Zero
		if model.TradeType == rabbitmq.TradeType_Buy {
			closeAmount := unitPrice.Mul(model.CloseAmount)
			equity = closeAmount
		} else {
			closeAmount := unitPrice.Mul(model.CloseAmount)
			openAmount := model.OpenPrice.Mul(model.CloseAmount)
			equity = openAmount.Sub(closeAmount).Add(openAmount)
			// *
			// * 例：當開倉賣100時, 關倉跌至80, 則賺20
			// * 保證金100, 加上20, 最後拿回120
			// * 最終拿回金額 = (開倉淨值 - 關倉淨值) + 保證金
			// * 目前保證金為開倉淨值
			// *
		}

		if balance.Add(equity).LessThan(decimal.Zero) {
			logging.Warn(ctx, "[MatchClosePosition] balance not enough: %v", common.ErrInsufficientBalance)
		}

		beforeAmount := balance.String()
		transactionRes, err := service.Impl.WalletIntf.Transaction(ctx, &wallet.TransactionReq{
			WalletID:     walletRes.Wallets[0].Id,
			Action:       wallet.Action_Action_CLOSE,
			Amount:       equity.String(),
			Currency:     productRes.Product.CurrencyCode,
			CommitterID:  model.MemberID,
			BeforeAmount: &beforeAmount,
		})
		if err != nil {
			logging.Warn(ctx, "[MatchClosePosition] Transaction failed. retry later: %v", err)
			continue
		}

		transactionID = transactionRes.Id
		deal = true
	}

	if !deal {
		logging.Error(ctx, "[MatchClosePosition] failed to match [%d]: %v", model.ID, common.ErrExceedRetryTimes)
		orderErr = common.ErrExceedRetryTimes
		return common.ErrExceedRetryTimes
	}

	if _, err := service.Impl.OrderIntf.UpdateOrderProcess(ctx, &order.UpdateOrderProcessReq{
		Id:           model.ID,
		OrderProcess: order.OrderProcess_OrderProcess_Matched,
	}); err != nil {
		logging.Error(ctx, "[MatchClosePosition] failed to Update OrderProcess [%d]: %v", model.ID, err)
	}

	if _, err := service.Impl.OrderIntf.FinishClosePositionOrder(ctx, &order.FinishClosePositionOrderReq{
		Id:                  model.ID,
		PositionID:          model.PositionID,
		UnitPrice:           unitPrice.String(),
		CloseAmount:         model.CloseAmount.String(),
		TransactionRecordID: uint64(transactionID),
		FinishedAt:          time.Now().Unix(),
	}); err != nil {
		logging.Error(ctx, "[MatchClosePosition] FinishClosePositionOrder failed: %v", err)
		if _, err := service.Impl.WalletIntf.RollbackTransaction(ctx, &wallet.RollbackTransactionReq{
			Id:           transactionID,
			RollbackerID: model.MemberID,
		}); err != nil {
			logging.Error(ctx, "[MatchClosePosition] failed to RollbackTransaction [%d]: %v", model.ID, err)
		}
		return err
	}

	matchRecord.MatchStatus = dbModels.MatchStatus_Finished
	closePrice = &decimal.NullDecimal{
		Valid:   true,
		Decimal: unitPrice,
	}

	orderProcess = order.OrderProcess_OrderProcess_Done
	expireTime := int64(time.Minute)
	expire = &expireTime

	return nil
}
