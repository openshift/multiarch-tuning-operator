package utils

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DecorateWithWaitGroup(wg *sync.WaitGroup, f func()) {
	wg.Add(1)
	go func() {
		defer GinkgoRecover()
		f()
		wg.Done()
	}()
}

func StartChannelMonitor(ch chan error, descr string) {
	go func() {
		defer GinkgoRecover()
		err := <-ch
		Expect(err).NotTo(HaveOccurred(), descr, err)
	}()
}

func EnsureNamespaces(ctx context.Context, client client.Client, namespaces ...string) {
	var err error
	for _, ns := range namespaces {
		namespace := &v1.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: ns,
			},
		}
		err = client.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
	}
}
