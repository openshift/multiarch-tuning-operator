package framework

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	ocpconfigv1 "github.com/openshift/api/config/v1"
)

func GetImageConfig(ctx context.Context, client runtimeclient.Client) (*ocpconfigv1.Image, error) {
	image := &ocpconfigv1.Image{}
	err := client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpconfigv1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}), image)
	if err != nil {
		return nil, err
	}
	return image, nil
}
