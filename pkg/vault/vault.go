package vault

import (
	"context"
	"errors"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"go.uber.org/zap"
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"
	"vault-injector/config"
	telegram "vault-injector/pkg"
)

type Service interface {
	IsNeedSecret(namespaceAndName string) bool
	GetData(ctx context.Context, namespace, name string) (map[string][]byte, error)
	GetSecretMap() SecretMap
	Start(ctx context.Context)
}

type vaultService struct {
	telegam      *telegram.Telegram
	secretMap    SecretMap
	cfg          *config.Config
	client       *vault.Client
	clientSecret *vault.Secret
	sync.Mutex
}

func NewVaultService(cfg *config.Config, telegram *telegram.Telegram) Service {
	return &vaultService{
		cfg:       cfg,
		telegam:   telegram,
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
func (v *vaultService) GetData(ctx context.Context, namespace, name string) (map[string][]byte, error) {
	secret, ok := v.secretMap[namespace+"/"+name]
	if !ok {
		return nil, nil
	}
	data := make(map[string][]byte)
	errFlag := false
	for _, vPath := range secret.ValuePath {
		_secretPath := strings.SplitN(vPath, ":", 3)
		_path := strings.SplitN(_secretPath[1], "/", 2)
		key := _secretPath[0]
		mount := _path[0]
		path := _path[1]
		vaultKey := _secretPath[2]
		ctx = context.WithValue(ctx, "secret", name+"("+namespace+")")
		secretData, err := v.GetVaultSecret(ctx, mount, path, vaultKey)
		if err != nil {
			data[key] = []byte{}
			errFlag = true
		}
		data[key] = secretData
	}
	if errFlag {
		return data, errors.New("get secret error")
	}
	return data, nil
}

func (v *vaultService) GetVaultSecret(ctx context.Context, mount, path, key string) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
	}()
	secretName := ctx.Value("secret").(string)
	secret, err := v.client.KVv2(mount).Get(ctx, path)
	zap.S().Debugf("%s getKV %s/%s:%s", secretName, mount, path, key)
	if err != nil {
		info := fmt.Sprintf("unable to read secret: %v", err)
		zap.S().Error(info)
		v.telegam.SendMessage(info)
		return nil, err
	}
	s := secret.Data[key]
	return []byte(fmt.Sprintf("%v", s)), nil
}

func vaultLogin(ctx context.Context, cfg *config.Config) (*vault.Client, *vault.Secret) {
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
		return nil, nil
	}

	authInfo, err := client.Auth().Login(ctx, k8sAuth)
	if err != nil {
		zap.S().Fatalf("una111ble to log in with Kubernetes auth: %v", err)
		return nil, nil
	}
	if authInfo == nil {
		zap.S().Fatalf("no auth info was returned after login")
		return nil, nil
	}
	return client, authInfo
}

func (v *vaultService) Start(ctx context.Context) {
	v.client, v.clientSecret = vaultLogin(ctx, v.cfg)
	v.initTelegram(ctx)
	zap.S().Infof("vault login success. duration: %d", v.clientSecret.Auth.LeaseDuration)
	go func() {
		zap.S().Info("vault started")
		ticker := time.NewTicker(time.Second*time.Duration(v.clientSecret.Auth.LeaseDuration) - 10*time.Second)
		for {
			select {
			case <-ctx.Done():
				zap.S().Info("finish main context")
				return
			case _ = <-ticker.C:
				v.client, v.clientSecret = vaultLogin(ctx, v.cfg)
			}
		}
	}()
}

func (v *vaultService) initTelegram(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Errorf("Init telegram failed: %v", err)
		}
	}()

	secret, err := v.client.KVv2("projects").Get(ctx, "share/telegram")
	zap.S().Debugf("%s getKV %s/%s", "telegram", "projects", "share/telegram")
	if err != nil {
		info := fmt.Sprintf("Init telegram failed: %v", err)
		zap.S().Error(info)
		return
	}
	ChatID, _ := strconv.ParseInt(secret.Data["channel"].(string), 10, 0)
	v.telegam.ChatID = ChatID
	v.telegam.Token = config.Password(secret.Data["token"].(string))
	zap.S().Info("Telegram initialized")
}
