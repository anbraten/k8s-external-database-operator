package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	anbratengithubiov1alpha1 "github.com/anbraten/k8s-external-database-operator/api/v1alpha1"
)

var _ = Describe("Database password", func() {
	ctx := context.Background()

	newDatabase := func(name string, password string, ref *anbratengithubiov1alpha1.SecretKeyReference) *anbratengithubiov1alpha1.Database {
		return &anbratengithubiov1alpha1.Database{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: anbratengithubiov1alpha1.DatabaseSpec{
				Type:              "postgres",
				Database:          "app",
				Username:          "app",
				Password:          password,
				PasswordSecretRef: ref,
			},
		}
	}

	ref := func(name, key string) *anbratengithubiov1alpha1.SecretKeyReference {
		return &anbratengithubiov1alpha1.SecretKeyReference{Name: name, Key: key}
	}

	Context("validation", func() {
		It("accepts a passwordSecretRef", func() {
			database := newDatabase("password-secret-ref", "", ref("app-db", "password"))
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			Expect(k8sClient.Delete(ctx, database)).To(Succeed())
		})

		It("rejects password and passwordSecretRef together", func() {
			err := k8sClient.Create(ctx, newDatabase("password-and-secret-ref", "secret", ref("app-db", "password")))
			Expect(err).To(MatchError(ContainSubstring("only one of password or passwordSecretRef may be set")))
		})

		It("rejects a passwordSecretRef without a key", func() {
			err := k8sClient.Create(ctx, newDatabase("password-secret-ref-without-key", "", ref("app-db", "")))
			Expect(err).To(MatchError(ContainSubstring("spec.passwordSecretRef.key")))
		})
	})

	Context("getPassword", func() {
		var reconciler *DatabaseReconciler

		BeforeEach(func() {
			reconciler = &DatabaseReconciler{Client: k8sClient, APIReader: k8sClient}
		})

		It("returns the inline password", func() {
			Expect(reconciler.getPassword(ctx, newDatabase("inline", "inline-secret", nil))).To(Equal("inline-secret"))
		})

		It("fails if no password is set", func() {
			_, err := reconciler.getPassword(ctx, newDatabase("no-password", "", nil))
			Expect(err).To(MatchError(ContainSubstring("neither password nor passwordSecretRef is set")))
		})

		Context("with a secret", func() {
			var secret *corev1.Secret

			BeforeEach(func() {
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "app-db", Namespace: "default"},
					Data: map[string][]byte{
						"password": []byte("from-secret"),
						"empty":    {},
					},
				}
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			})

			It("loads the password from the secret", func() {
				Expect(reconciler.getPassword(ctx, newDatabase("from-secret", "", ref("app-db", "password")))).To(Equal("from-secret"))
			})

			It("fails if the key is missing", func() {
				_, err := reconciler.getPassword(ctx, newDatabase("missing-key", "", ref("app-db", "nope")))
				Expect(err).To(MatchError(ContainSubstring("secret 'app-db' has no key 'nope'")))
			})

			It("fails if the value is empty", func() {
				_, err := reconciler.getPassword(ctx, newDatabase("empty-key", "", ref("app-db", "empty")))
				Expect(err).To(MatchError(ContainSubstring("key 'empty' of secret 'app-db' is empty")))
			})

			It("only reads secrets from the namespace of the database", func() {
				database := newDatabase("other-namespace", "", ref("app-db", "password"))
				database.Namespace = "kube-system"
				_, err := reconciler.getPassword(ctx, database)
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})

		It("fails if the secret is missing", func() {
			_, err := reconciler.getPassword(ctx, newDatabase("missing-secret", "", ref("does-not-exist", "password")))
			Expect(err).To(MatchError(ContainSubstring("failed to get secret 'does-not-exist'")))
		})
	})
})
