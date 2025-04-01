package framework

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRandomNodeName(ctx context.Context, client runtimeclient.Client, labelKey string, labelInValue string) (string, error) {
	nodes, err := GetNodesWithLabel(ctx, client, labelKey, labelInValue)
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("got null nodes by key %s and value %s lable", labelKey, labelInValue)
	}
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(nodes.Items))))
	if err != nil {
		return "", err
	}
	return nodes.Items[randomIndex.Int64()].Name, nil
}

func GetNodesWithLabel(ctx context.Context, client runtimeclient.Client, labelKey string, labelInValue string) (*corev1.NodeList, error) {
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
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*r)
	nodes := &corev1.NodeList{}
	err = client.List(ctx, nodes, &runtimeclient.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
