package framework

import (
	"context"
	"crypto/rand"
	"math/big"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRandomNodeName(ctx context.Context, client runtimeclient.Client, labelKey string, labelInValue string) (string, error) {
	var (
		r   *labels.Requirement
		err error
	)
	if labelInValue == "" {
		r, err = labels.NewRequirement(labelKey, selection.Exists, nil)
	} else {
		r, err = labels.NewRequirement(labelKey, selection.In, []string{labelInValue})
	}
	if err != nil {
		return "", err
	}
	labelSelector := labels.NewSelector().Add(*r)
	workerNodes := &corev1.NodeList{}
	err = client.List(ctx, workerNodes, &runtimeclient.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", err
	}
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(workerNodes.Items))))
	if err != nil {
		return "", err
	}
	return workerNodes.Items[randomIndex.Int64()].Name, nil
}
