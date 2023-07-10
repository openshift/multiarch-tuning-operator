package image

import (
	"encoding/json"
	"errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func ExtractAuthFromSecret(secret *v1.Secret) ([]byte, error) {
	switch secret.Type {
	case "kubernetes.io/dockercfg":
		return secret.Data[".dockercfg"], nil
	case "kubernetes.io/dockerconfigjson":
		var objmap map[string]json.RawMessage
		if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &objmap); err != nil {
			klog.Warningf("Error unmarshaling secret data for: %s/%s", secret.Namespace, secret.Name)
			return nil, err
		}
		return objmap["auths"], nil
	}
	return nil, errors.New("unknown secret type")
}
