package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	anbratengithubiov1alpha1 "github.com/anbraten/k8s-external-database-operator/api/v1alpha1"
)

var _ = Describe("Database validation", func() {
	newDatabase := func(name string, databaseType string, extensions []string) *anbratengithubiov1alpha1.Database {
		return &anbratengithubiov1alpha1.Database{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: anbratengithubiov1alpha1.DatabaseSpec{
				Type:     databaseType,
				Database: "app",
				Username: "app",
				Password: "secret",
				Postgres: &anbratengithubiov1alpha1.PostgresSpec{Extensions: extensions},
			},
		}
	}

	It("accepts postgres settings for type postgres", func() {
		database := newDatabase("postgres-with-extensions", "postgres", []string{"vector", "uuid-ossp"})
		Expect(k8sClient.Create(context.Background(), database)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), database)).To(Succeed())
	})

	It("rejects postgres settings for other types", func() {
		err := k8sClient.Create(context.Background(), newDatabase("mysql-with-extensions", "mysql", []string{"vector"}))
		Expect(err).To(MatchError(ContainSubstring("postgres can only be set if type is postgres")))
	})

	It("rejects invalid extension names", func() {
		err := k8sClient.Create(context.Background(), newDatabase("invalid-extension", "postgres", []string{"vector; DROP"}))
		Expect(err).To(MatchError(ContainSubstring("should match")))
	})
})
