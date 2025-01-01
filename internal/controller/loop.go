package controller

import (
	"context"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"time"
	"vault-injector/config"
	"vault-injector/internal/k8s"
	"vault-injector/pkg/vault"
)

type loopController struct {
	p loopControllerParams
}

type loopControllerParams struct {
	dig.In

	Cfg   *config.Config
	Kr    k8s.KubeRepo
	Vault vault.Service
}

type LoopController interface {
	UpdateSecretList(ctx context.Context)
	Start(ctx context.Context)
}

func (c *loopController) UpdateSecretList(ctx context.Context) {
	zap.S().Infof("UpdateSecretList start")
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
		zap.S().Infof("%s UpdateSecretList finish", time.Now())
	}()

	secretList := c.p.Kr.GetSecretList(ctx)
	secretMap := c.p.Vault.GetSecretMap()
	for _, secret := range secretList.Items {
		if _, ok := secretMap[secret.Namespace+"/"+secret.Name]; ok {
			delete(secretMap, secret.Namespace+"/"+secret.Name)
		}
	}
	for _, newSecret := range secretMap {
		zap.S().Info("%s(%s) create empty secret", newSecret.Name, newSecret.Namespace)
		c.p.Kr.CreateEmptySecret(ctx, newSecret.Namespace, newSecret.Name)
	}
	for _, secret := range secretList.Items {
		c.p.Kr.CompareSecret(ctx, &secret)
	}

}

func (c *loopController) Start(ctx context.Context) {
	go func() {
		zap.S().Info("LoopController start")
		c.UpdateSecretList(ctx)
		ticker := time.NewTicker(time.Second * time.Duration(c.p.Cfg.Interval))
		for {
			select {
			case <-ctx.Done():
				zap.S().Info("finish main context")
				return
			case _ = <-ticker.C:
				c.UpdateSecretList(ctx)
			}
		}
	}()
}

func NewLoopController(p loopControllerParams) Result {
	return Result{
		Controller: &loopController{
			p: p,
		},
	}
}
