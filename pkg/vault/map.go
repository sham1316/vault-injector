package vault

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type _SecretMap map[string][]string

type Secret struct {
	Namespace string
	Name      string
	ValuePath []string
}

type SecretMap map[string]Secret

func ParseMap(file string) SecretMap {
	zap.S().Debug(file)
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		zap.S().Errorf("yamlFile.Get err   #%v ", err)
	}
	var _secretMap _SecretMap
	err = yaml.Unmarshal(yamlFile, &_secretMap)
	if err != nil {
		zap.S().Errorf("Unmarshal: %v", err)
	}
	var secrets = make(SecretMap)
	for k, v := range _secretMap {
		_ss := strings.Split(k, "/")
		s := Secret{
			Namespace: _ss[0],
			Name:      _ss[1],
			ValuePath: v,
		}
		secrets[k] = s
	}
	return secrets
}
