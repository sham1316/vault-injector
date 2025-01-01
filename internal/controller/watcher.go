package controller

import (
	"context"
	"go.uber.org/dig"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"reflect"
	"sync"
	"time"
	"vault-injector/config"
	"vault-injector/internal/k8s"
)

type watchControllerParams struct {
	dig.In

	Cfg *config.Config
	Kr  k8s.KubeRepo
}

type watchController struct {
	p watchControllerParams
}

func (w *watchController) Watch(ctx context.Context) {
	watcher := w.p.Kr.WatchSecretList(ctx)
	if watcher == nil {
		return
	}
	zap.S().Info("WatchController start")
	defer watcher.Stop()
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				//the channel got closed, so we need to restart
				zap.S().Warnf("WatchController hung up on us, need restart event watcher")
				return
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				secret, ok := event.Object.(*v1.Secret)
				if !ok {
					zap.S().Errorf("unexpected type %s, %+v", reflect.TypeOf(event.Object), event)
				} else {
					zap.S().Infof("%s(%s) added or modified", secret.Name, secret.Namespace)
					w.p.Kr.CompareSecret(ctx, secret)
				}
			}
		case <-ctx.Done():
			zap.S().Infof("Exit from Watcher because the context is done")
			return
		}
	}
}

func (w *watchController) Start(ctx context.Context) {
	go func() {
		var wg sync.WaitGroup
		for {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if ctx.Err() == nil {
					w.Watch(ctx)
				}
			}()
			wg.Wait()
			time.Sleep(1 * time.Second)
		}
	}()
}

func NewWatchController(p watchControllerParams) Result {
	return Result{
		Controller: &watchController{
			p: p,
		},
	}
}
