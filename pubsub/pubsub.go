package pubsub

import (
	"context"
	"reflect"
	"sync"

	"github.com/paper-trade-chatbot/be-common/config"
	"github.com/paper-trade-chatbot/be-common/logging"
	"github.com/paper-trade-chatbot/be-match/pubsub/matchClosePosition"
	"github.com/paper-trade-chatbot/be-match/pubsub/matchOpenPosition"
	bePubsub "github.com/paper-trade-chatbot/be-pubsub"

	rabbitmqClosePosition "github.com/paper-trade-chatbot/be-pubsub/order/closePosition/rabbitmq"
	rabbitmqOpenPosition "github.com/paper-trade-chatbot/be-pubsub/order/openPosition/rabbitmq"
)

var publishers = map[string]interface{}{}
var publisherLock sync.RWMutex
var subscribers = []bePubsub.Pubsub{}

//Initialize
// please register all instance creator of publisher here
func Initialize(ctx context.Context) {

	bePubsub.LogMode = true

	// ==============================
	// |   initialize publishers    |
	// ==============================

	func() {
		publisherLock.Lock()
		defer publisherLock.Unlock()

	}()

	for k, v := range publishers {
		logging.Info(ctx, "publisher [%s] [*%s] initialized.", k, reflect.TypeOf(v).Elem().Name())
	}

	// ==============================
	// |    register subscribers    |
	// ==============================

	if sub, err := rabbitmqOpenPosition.SubscribeAndListen(
		ctx,
		config.GetString("RABBITMQ_USERNAME"),
		config.GetString("RABBITMQ_PASSWORD"),
		config.GetString("RABBITMQ_HOST"),
		config.GetString("RABBITMQ_VIRTUAL_HOST"),
		config.GetString("SERVICE_NAME"),
		matchOpenPosition.MatchOpenPosition,
	); err != nil {
		logging.Error(ctx, "SubscribeAndListen error %v", err)
		panic(err)
	} else {
		subscribers = append(subscribers, sub)
	}

	if sub, err := rabbitmqClosePosition.SubscribeAndListen(
		ctx,
		config.GetString("RABBITMQ_USERNAME"),
		config.GetString("RABBITMQ_PASSWORD"),
		config.GetString("RABBITMQ_HOST"),
		config.GetString("RABBITMQ_VIRTUAL_HOST"),
		config.GetString("SERVICE_NAME"),
		matchClosePosition.MatchClosePosition,
	); err != nil {
		logging.Error(ctx, "SubscribeAndListen error %v", err)
		panic(err)
	} else {
		subscribers = append(subscribers, sub)
	}

}

func Finalize(ctx context.Context) {

	for _, s := range subscribers {
		s.Close()
	}

	publisherLock.Lock()
	defer publisherLock.Unlock()
	for k, v := range publishers {
		err := v.(bePubsub.Pubsub).Close()
		if err != nil {
			logging.Error(ctx, "pubsub Finalize error %v", err)
		}
		delete(publishers, k)
	}
}

func GetPublisher[T interface{}](ctx context.Context) bePubsub.TPublisher[T] {

	if !publisherLock.TryRLock() {
		logging.Error(ctx, "GetPublisher: not initialized yet.")
		return nil
	}
	defer publisherLock.RUnlock()

	var model T
	modelType := reflect.TypeOf(model).String()

	if _, ok := publishers[modelType]; !ok {
		logging.Error(ctx, "GetPublisher: no such publisher.", modelType)
		return nil
	}
	return publishers[modelType].(bePubsub.TPublisher[T])
}

func registerPublisher[T interface{}](publisher bePubsub.TPublisher[T]) error {
	var model T
	publishers[reflect.TypeOf(model).String()] = publisher
	return nil
}
