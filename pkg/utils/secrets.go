package utils

import (
	"encoding/json"
	"errors"
	"k8s.io/api/core/v1"
)

func ExtractAuthFromSecret(secret *v1.Secret) ([]byte, error) {
	switch secret.Type {
	case "kubernetes.io/dockercfg":
		return secret.Data[".dockercfg"], nil
	case "kubernetes.io/dockerconfigjson":
		var objmap map[string]json.RawMessage
		if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &objmap); err != nil {
			return nil, err
		}
		return objmap["auths"], nil
	}
	return nil, errors.New("unknown secret type")
}
