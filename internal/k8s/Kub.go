package k8s

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"reflect"
	"strings"
	"vault-injector/config"
	"vault-injector/pkg/vault"
)

type KubeRepo interface {
	DeleteSecret(ctx context.Context, namespace, name string)
	CreateEmptySecret(ctx context.Context, namespace, name string)
	CreateDockerSecret(ctx context.Context, namespace, name string)
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

func (kr *kubeRepo) _newSecret(namespace, name string, _type v1.SecretType) *v1.Secret {
	labels := map[string]string{
		kr.cfg.SecretLabel + "/sync": "true",
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: _type,
	}
}

func (kr *kubeRepo) NewSecret(namespace, name string) *v1.Secret {
	return kr._newSecret(namespace, name, v1.SecretTypeOpaque)
}

func (kr *kubeRepo) NewDockerSecret(namespace, name string, data map[string][]byte) *v1.Secret {
	secret := kr._newSecret(namespace, name, v1.SecretTypeDockerConfigJson)
	secret.Data = data
	return secret
}

func (kr *kubeRepo) CompareSecret(ctx context.Context, secret *v1.Secret) {
	var data map[string][]byte
	var err error
	if strings.Contains(secret.Name, "dockerconfigjson") {
		data, err = kr.vault.GetDockerData(ctx, secret.Namespace, secret.Name)
	} else {
		data, err = kr.vault.GetData(ctx, secret.Namespace, secret.Name)
	}

	if data == nil {
		zap.S().Infof("%s(%s) no in secretMap - DELETE", secret.Namespace, secret.Name)
		kr.DeleteSecret(ctx, secret.Namespace, secret.Name)
		return
	}
	if err != nil {
		zap.S().Infof("%s(%s) GetSecret error - SKIP", secret.Namespace, secret.Name)
		return
	}
	info := fmt.Sprintf("%s(%s) check for update", secret.Namespace, secret.Name)
	if reflect.DeepEqual(secret.Data, data) {
		zap.S().Infof("%s - EQUALS", info)
	} else {
		secret.Data = data
		zap.S().Infof("%s - NOT EQUALS", info)
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

func (kr *kubeRepo) CreateDockerSecret(ctx context.Context, namespace, name string) {
	data, err := kr.vault.GetDockerData(ctx, namespace, name)
	secret := kr.NewDockerSecret(namespace, name, data)
	err = kr.ks.CreateSecret(ctx, secret)
	if err != nil {
		zap.S().Errorf("error CreateEmptyDockerSecret: %v", err)
	}
}
