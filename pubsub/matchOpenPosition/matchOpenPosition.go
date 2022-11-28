package matchOpenPosition

import (
	"context"
	"time"

	"github.com/paper-trade-chatbot/be-match/dao/matchRecordDao"
	"github.com/paper-trade-chatbot/be-match/database"
	"github.com/paper-trade-chatbot/be-match/logging"
	"github.com/paper-trade-chatbot/be-match/models/dbModels"
	"github.com/paper-trade-chatbot/be-match/service"
	"github.com/paper-trade-chatbot/be-proto/order"
	"github.com/paper-trade-chatbot/be-proto/product"
	"github.com/paper-trade-chatbot/be-proto/quote"
	"github.com/paper-trade-chatbot/be-proto/wallet"
	"github.com/paper-trade-chatbot/be-pubsub/order/openPosition/rabbitmq"
	"github.com/shopspring/decimal"
)

func MatchOpenPosition(ctx context.Context, model *rabbitmq.OpenPositionModel) error {
	db := database.GetDB()
	deal := false
	retryCount := 0
	var transactionID uint64 = 0
	unitPrice := decimal.Decimal{}

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
	}

	defer func() {
		if matchRecord.MatchStatus != dbModels.MatchStatus_Finished {
			matchRecord.MatchStatus = dbModels.MatchStatus_Failed
		}

		if err := matchRecordDao.Modify(db, matchRecord, &matchRecordDao.UpdateModel{
			MatchStatus: &matchRecord.MatchStatus,
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
		if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
			Id: model.ID,
		}); err != nil {
			logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
		}
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
			logging.Error(ctx, "[MatchOpenPosition] failed to get wallet by member[%d] currency[%s]: %v", model.MemberID, productRes.Product.CurrencyCode, err)
			if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
				Id: model.ID,
			}); err != nil {
				logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
			}
			return err
		}

		balance, err := decimal.NewFromString(walletRes.Wallets[0].Amount)
		if err != nil {
			logging.Error(ctx, "[MatchOpenPosition] NewFromString failed: %v", err)
			if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
				Id: model.ID,
			}); err != nil {
				logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
			}
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
			logging.Error(ctx, "[MatchOpenPosition] balance not enough: %v", err)
			if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
				Id: model.ID,
			}); err != nil {
				logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
			}
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
		logging.Error(ctx, "[MatchOpenPosition] failed to match [%d]: %v", model.ID, err)
		if _, err := service.Impl.OrderIntf.FailOrder(ctx, &order.FailOrderReq{
			Id: model.ID,
		}); err != nil {
			logging.Error(ctx, "[MatchOpenPosition] failed to FailOrder [%d]: %v", model.ID, err)
		}
		return err
	}

	if _, err := service.Impl.OrderIntf.FinishOpenPositionOrder(ctx, &order.FinishOpenPositionOrderReq{
		Id:                  model.ID,
		UnitPrice:           unitPrice.String(),
		TransactionRecordID: uint64(transactionID),
		FinishedAt:          time.Now().Unix(),
	}); err != nil {
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

	return nil
}
