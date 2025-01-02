package k8s

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"reflect"
	"vault-injector/config"
	"vault-injector/pkg/vault"
)

type KubeRepo interface {
	DeleteSecret(ctx context.Context, namespace, name string)
	CreateEmptySecret(ctx context.Context, namespace, name string)
	UpdateSecret(ctx context.Context, secret *v1.Secret)
	GetSecretList(ctx context.Context) *v1.SecretList
	WatchSecretList(ctx context.Context) watch.Interface
	CompareSecret(ctx context.Context, secret *v1.Secret)
}

type kubeRepo struct {
	cfg   *config.Config
	ks    KubeService
	vault vault.Service
}

func NewKubeRepo(ks KubeService, cfg *config.Config, vault vault.Service) KubeRepo {
	return &kubeRepo{
		cfg:   cfg,
		ks:    ks,
		vault: vault,
	}
}

func (kr *kubeRepo) GetSecretList(ctx context.Context) *v1.SecretList {
	serviceList, err := kr.ks.GetSecretList(ctx)
	if err != nil {
		zap.S().Errorf("error GetSecretList: %v", err)
		return nil
	}
	return serviceList
}

func (kr *kubeRepo) WatchSecretList(ctx context.Context) watch.Interface {
	serviceList, err := kr.ks.WatchSecretList(ctx)
	if err != nil {
		zap.S().Errorf("error WatchSecretList: %v", err)
		return nil
	}
	return serviceList
}

func (kr *kubeRepo) DeleteSecret(ctx context.Context, namespace, name string) {
	err := kr.ks.DeleteSecret(ctx, namespace, name)
	if err != nil {
		zap.S().Errorf("error GetSecretList: %v", err)
	}
}

func (kr *kubeRepo) UpdateSecret(ctx context.Context, secret *v1.Secret) {
	err := kr.ks.UpdateSecret(ctx, secret)
	if err != nil {
		zap.S().Errorf("error UpdateSecret: %v", err)
	}
}

func (kr *kubeRepo) NewSecret(namespace, name string) *v1.Secret {
	labels := map[string]string{
		kr.cfg.SecretLabel + "/sync": "true",
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: v1.SecretTypeOpaque,
	}
}

func (kr *kubeRepo) CompareSecret(ctx context.Context, secret *v1.Secret) {
	data, err := kr.vault.GetData(ctx, secret.Namespace, secret.Name)
	if data == nil {
		zap.S().Infof("%s(%s) no in secretMap - delete", secret.Namespace, secret.Name)
		kr.DeleteSecret(ctx, secret.Namespace, secret.Name)
		return
	}
	if err != nil {
		zap.S().Infof("%s(%s) GetSecret error - skip", secret.Namespace, secret.Name)
		return
	}
	info := fmt.Sprintf("%s(%s) check for update", secret.Namespace, secret.Name)
	if reflect.DeepEqual(secret.Data, data) {
		zap.S().Infof("%s - equals", info)
	} else {
		secret.Data = data
		zap.S().Infof("%s - not equals", info)
		kr.UpdateSecret(ctx, secret)
	}
}

func (kr *kubeRepo) CreateEmptySecret(ctx context.Context, namespace, name string) {
	secret := kr.NewSecret(namespace, name)
	err := kr.ks.CreateSecret(ctx, secret)
	if err != nil {
		zap.S().Errorf("error CreateEmptySecret: %v", err)
	}

}
