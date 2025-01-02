package vault

import (
	"context"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"go.uber.org/zap"
	"maps"
	"strings"
	"sync"
	"time"
	"vault-injector/config"
)

type Service interface {
	IsNeedSecret(namespaceAndName string) bool
	GetData(ctx context.Context, namespace, name string) map[string][]byte
	GetSecretMap() SecretMap
	Start(ctx context.Context)
}

type vaultService struct {
	secretMap SecretMap
	cfg       *config.Config
	client    *vault.Client
	sync.Mutex
}

func NewVaultService(cfg *config.Config) Service {
	return &vaultService{
		cfg:       cfg,
		secretMap: ParseMap(cfg.SecretMap),
	}
}

func (v *vaultService) IsNeedSecret(namespaceAndName string) bool {
	_, ok := v.secretMap[namespaceAndName]
	return ok
}

func (v *vaultService) GetSecretMap() SecretMap {
	v.Lock()
	defer v.Unlock()
	return maps.Clone(v.secretMap)
}
func (v *vaultService) GetData(ctx context.Context, namespace, name string) map[string][]byte {
	secret, ok := v.secretMap[namespace+"/"+name]
	if !ok {
		return nil
	}
	data := make(map[string][]byte)
	for _, path := range secret.ValuePath {
		_secretPath := strings.SplitN(path, ":", 3)
		_path := strings.SplitN(_secretPath[1], "/", 2)
		key := _secretPath[0]
		mount := _path[0]
		path := _path[1]
		vaultKey := _secretPath[2]
		ctx = context.WithValue(ctx, "secret", name+"("+namespace+")")
		secretData := v.GetVaultSecret(ctx, mount, path, vaultKey)
		data[key] = secretData
	}
	return data
}

func (v *vaultService) GetVaultSecret(ctx context.Context, mount, path, key string) []byte {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
	}()
	secretName := ctx.Value("secret").(string)
	secret, err := v.client.KVv2(mount).Get(ctx, path)
	zap.S().Debugf("%s getKV %s/%s:%s", secretName, mount, path, key)
	if err != nil {
		zap.S().Errorf("unable to read secret: %v", err)
		return nil
	}
	s := secret.Data[key]
	return []byte(fmt.Sprintf("%v", s))
}

func vaultLogin(ctx context.Context, cfg *config.Config) *vault.Client {
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = cfg.VaultAddr
	vaultConfig.Timeout = 60 * time.Second
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		zap.S().Fatalf("can`t create vault client: %v", err)

	}

	k8sAuth, err := auth.NewKubernetesAuth(
		cfg.VaultRole,
		auth.WithServiceAccountTokenPath(cfg.TokenPath))

	if err != nil {
		zap.S().Fatalf("unable to initialize Kubernetes auth method: %v", err)
		return nil
	}

	authInfo, err := client.Auth().Login(ctx, k8sAuth)
	if err != nil {
		zap.S().Fatalf("una111ble to log in with Kubernetes auth: %v", err)
		return nil
	}
	if authInfo == nil {
		zap.S().Fatalf("no auth info was returned after login")
		return nil
	}
	zap.S().Infof("vault login success. duration:	 %d", authInfo.LeaseDuration)
	return client
}

func (v *vaultService) Start(ctx context.Context) {
	v.client = vaultLogin(ctx, v.cfg)
	go func() {
		zap.S().Info("vault start")
		ticker := time.NewTicker(time.Second*time.Duration(v.cfg.Interval) - 10*time.Second)
		for {
			select {
			case <-ctx.Done():
				zap.S().Info("finish main context")
				return
			case _ = <-ticker.C:
				v.client = vaultLogin(ctx, v.cfg)
			}
		}
	}()
}
