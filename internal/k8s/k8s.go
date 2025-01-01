package k8s

import (
	"context"
	"flag"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sync"
	"vault-injector/config"
)

type KubeService interface {
	GetSecretList(ctx context.Context) (*v1.SecretList, error)
	CreateSecret(ctx context.Context, secret *v1.Secret) error
	UpdateSecret(ctx context.Context, secret *v1.Secret) error
	DeleteSecret(ctx context.Context, namespace, name string) error
	WatchSecretList(ctx context.Context) (watch.Interface, error)
	GetToken() string
	GetCA() []byte
}

type kubeService struct {
	Cfg             *config.Config
	k8sConfig       *rest.Config
	clientSet       *kubernetes.Clientset
	resourceVersion string
	labelSelector   *metav1.LabelSelector
	sync.Mutex
}

func NewKubeService(cfg *config.Config) KubeService {
	k8sConfig := getConfig(cfg.InCluster, cfg.Kubeconfig)
	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		zap.S().Fatal(err)
	}
	labelSelector := &metav1.LabelSelector{MatchLabels: map[string]string{cfg.SecretLabel + "/sync": "true"}}
	return &kubeService{
		Cfg:             cfg,
		k8sConfig:       k8sConfig,
		clientSet:       clientSet,
		labelSelector:   labelSelector,
		resourceVersion: "0",
	}
}

func getConfig(inCluster bool, kubeconfig string) *rest.Config {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = *flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
			} else {
				kubeconfig = *flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
			}
			flag.Parse()
		}
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		zap.S().Fatal(err)
	}
	return config
}
func (k *kubeService) GetToken() string {
	return k.k8sConfig.BearerToken
}

func (k *kubeService) GetCA() []byte {
	return k.k8sConfig.TLSClientConfig.CAData
}

func (k *kubeService) GetSecretList(ctx context.Context) (*v1.SecretList, error) {
	k.Lock()
	defer k.Unlock()
	opt := metav1.ListOptions{LabelSelector: labels.Set(k.labelSelector.MatchLabels).String()}
	n, err := k.clientSet.CoreV1().Secrets("").List(ctx, opt)
	k.resourceVersion = n.GetResourceVersion()
	return n, err
}

func (k *kubeService) WatchSecretList(ctx context.Context) (watch.Interface, error) {
	k.Lock()
	defer k.Unlock()
	opt := metav1.ListOptions{LabelSelector: labels.Set(k.labelSelector.MatchLabels).String(), ResourceVersion: k.resourceVersion}
	w, err := k.clientSet.CoreV1().Secrets("").Watch(ctx, opt)
	return w, err
}

func (k *kubeService) DeleteSecret(ctx context.Context, namespace, name string) error {
	k.Lock()
	defer k.Unlock()
	err := k.clientSet.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func (k *kubeService) UpdateSecret(ctx context.Context, secret *v1.Secret) error {
	k.Lock()
	defer k.Unlock()
	_, err := k.clientSet.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func (k *kubeService) CreateSecret(ctx context.Context, secret *v1.Secret) error {
	k.Lock()
	defer k.Unlock()
	_, err := k.clientSet.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}
