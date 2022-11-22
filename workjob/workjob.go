package workjob

import (
	"context"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/paper-trade-chatbot/be-match/logging"
)

type Job struct {
	Workjob func(context.Context) error
	Cancel  context.CancelFunc
}

var jobs map[string]*Job

func Initialize(ctx context.Context) {

	for _, j := range jobs {
		go work(ctx, j)
	}
}

func Finalize(ctx context.Context) {
	for _, j := range jobs {
		if j != nil {
			j.Cancel()
		}
	}
}

func work(ctx context.Context, job *Job) {

	workjobID, _ := uuid.NewV4()
	ctxWithValue := context.WithValue(ctx, logging.ContextKeyRequestId, workjobID.String())

	funcName := strings.Split(runtime.FuncForPC(reflect.ValueOf(job.Workjob).Pointer()).Name(), "/")
	logging.Info(ctxWithValue, "[Workjob] start %s", funcName[len(funcName)-1])
	key := "Workjob:" + funcName[len(funcName)-1]

	ctxCancel, cancel := context.WithCancel(ctxWithValue)
	defer cancel()
	job.Cancel = cancel

	select {
	case <-ctxCancel.Done():
		logging.Error(ctxCancel, "[Workjob] %s cancelled error: %v", key, ctxCancel.Err())
	default:
		func(ctxCancel context.Context) {
			defer func() {
				if r := recover(); r != nil {
					logging.Error(ctxCancel, "\x1b[31m%v\n[Stack Trace]\n%s\x1b[m", r, debug.Stack())
				}
			}()
			err := job.Workjob(ctxCancel)
			if err != nil {
				logging.Error(ctxCancel, "[Workjob] %s error: %v", key, err)
			}
		}(ctxCancel)
	}
	logging.Info(ctxWithValue, "[Workjob] %s end", funcName[len(funcName)-1])
}
