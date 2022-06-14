package test

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	repoURL = os.Getenv("REPO_URL") // xx.amazonaws.com/foo
)

const (
	resultWaitTimeout = 20 * time.Second
	operationTimeout  = 10 * time.Second
)

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestImageRepositoryScan(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()

	repo := &imagev1.ImageRepository{
		Spec: imagev1.ImageRepositorySpec{
			Interval: v1.Duration{Duration: 30 * time.Second},
			Image:    repoURL,
		},
	}
	repoObjectKey := types.NamespacedName{
		Name:      "test-repo-" + randStringRunes(5),
		Namespace: "default",
	}
	repo.Name = repoObjectKey.Name
	repo.Namespace = repoObjectKey.Namespace

	g.Expect(kubeClient.Create(ctx, repo)).To(Succeed())
	defer func() {
		g.Expect(kubeClient.Delete(ctx, repo)).To(Succeed())
	}()
	g.Eventually(func() bool {
		if err := kubeClient.Get(ctx, repoObjectKey, repo); err != nil {
			return false
		}
		return repo.Status.LastScanResult != nil
	}, resultWaitTimeout).Should(BeTrue())
	g.Expect(repo.Status.CanonicalImageName).To(Equal(repoURL))
	g.Expect(repo.Status.LastScanResult.TagCount > 0).To(BeTrue())
}
