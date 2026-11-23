package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

	newSecret := func(name string, data map[string]string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			StringData: data,
		}
	}

	Context("validation", func() {
		It("accepts neither password nor passwordSecretRef", func() {
			database := newDatabase("no-password", "", nil)
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			Expect(k8sClient.Delete(ctx, database)).To(Succeed())
		})

		It("accepts only a password", func() {
			database := newDatabase("only-password", "secret", nil)
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			Expect(k8sClient.Delete(ctx, database)).To(Succeed())
		})

		It("accepts only a passwordSecretRef", func() {
			database := newDatabase("only-password-secret-ref", "", ref("app-db", "password"))
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			Expect(k8sClient.Delete(ctx, database)).To(Succeed())
		})

		It("rejects password and passwordSecretRef together", func() {
			err := k8sClient.Create(ctx, newDatabase("password-and-secret-ref", "secret", ref("app-db", "password")))
			Expect(err).To(MatchError(ContainSubstring("only one of password or passwordSecretRef may be set")))
		})

		It("rejects adding a passwordSecretRef to a database with a password", func() {
			database := newDatabase("add-secret-ref", "secret", nil)
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, database)

			database.Spec.PasswordSecretRef = ref("app-db", "password")
			err := k8sClient.Update(ctx, database)
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

		It("returns the inline password with the generation as version", func() {
			database := newDatabase("inline", "inline-secret", nil)
			database.Generation = 3
			password, version, err := reconciler.getPassword(ctx, database)
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(Equal("inline-secret"))
			Expect(version).To(Equal("spec/3"))
		})

		It("returns no password if none is set", func() {
			password, version, err := reconciler.getPassword(ctx, newDatabase("no-password", "", nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(BeEmpty())
			Expect(version).To(BeEmpty())
		})

		It("fails if password and passwordSecretRef are set (resources created before the validation)", func() {
			_, _, err := reconciler.getPassword(ctx, newDatabase("both", "inline-secret", ref("app-db", "password")))
			Expect(err).To(MatchError(ContainSubstring("only one of password or passwordSecretRef may be set")))
		})

		It("fails if the secret is missing", func() {
			_, _, err := reconciler.getPassword(ctx, newDatabase("missing-secret", "", ref("does-not-exist", "password")))
			Expect(err).To(MatchError(ContainSubstring("failed to get secret 'does-not-exist'")))
		})

		Context("with a secret", func() {
			var secret *corev1.Secret

			BeforeEach(func() {
				secret = newSecret("app-db", map[string]string{"password": "from-secret", "other": "other-secret", "empty": ""})
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				DeferCleanup(k8sClient.Delete, ctx, secret)
			})

			It("loads the password from the secret", func() {
				password, version, err := reconciler.getPassword(ctx, newDatabase("from-secret", "", ref("app-db", "password")))
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(Equal("from-secret"))
				Expect(version).To(Equal(fmt.Sprintf("secret/app-db/password/%s/%s", secret.UID, secret.ResourceVersion)))
			})

			It("changes the version if the secret or the referenced key changes", func() {
				database := newDatabase("from-secret", "", ref("app-db", "password"))
				_, version, err := reconciler.getPassword(ctx, database)
				Expect(err).NotTo(HaveOccurred())

				database.Spec.PasswordSecretRef.Key = "other"
				password, otherKeyVersion, err := reconciler.getPassword(ctx, database)
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(Equal("other-secret"))
				Expect(otherKeyVersion).NotTo(Equal(version))

				secret.StringData = map[string]string{"other": "changed"}
				Expect(k8sClient.Update(ctx, secret)).To(Succeed())
				password, changedVersion, err := reconciler.getPassword(ctx, database)
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(Equal("changed"))
				Expect(changedVersion).NotTo(Equal(otherKeyVersion))
			})

			It("fails if the key is missing", func() {
				_, _, err := reconciler.getPassword(ctx, newDatabase("missing-key", "", ref("app-db", "nope")))
				Expect(err).To(MatchError(ContainSubstring("secret 'app-db' has no key 'nope'")))
			})

			It("fails if the value is empty", func() {
				_, _, err := reconciler.getPassword(ctx, newDatabase("empty-key", "", ref("app-db", "empty")))
				Expect(err).To(MatchError(ContainSubstring("key 'empty' of secret 'app-db' is empty")))
			})

			It("only reads secrets from the namespace of the database", func() {
				database := newDatabase("other-namespace", "", ref("app-db", "password"))
				database.Namespace = "kube-system"
				_, _, err := reconciler.getPassword(ctx, database)
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})
	})

	Context("databasesForSecret", func() {
		It("returns the databases referencing the secret in the same namespace", func() {
			referencing := newDatabase("referencing", "", ref("app-db", "password"))
			otherSecret := newDatabase("other-secret", "", ref("other-db", "password"))
			inline := newDatabase("inline", "secret", nil)
			otherNamespace := newDatabase("other-namespace", "", ref("app-db", "password"))
			otherNamespace.Namespace = "other"

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(referencing, otherSecret, inline, otherNamespace).
				WithIndex(&anbratengithubiov1alpha1.Database{}, passwordSecretRefIndex, indexPasswordSecretRef).
				Build()
			reconciler := &DatabaseReconciler{Client: fakeClient, Log: GinkgoLogr}

			secret := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "app-db", Namespace: "default"}}
			Expect(reconciler.databasesForSecret(ctx, secret)).To(ConsistOf(
				reconcile.Request{NamespacedName: client.ObjectKeyFromObject(referencing)},
			))
		})
	})

	// Requires the postgres database of the docker-compose.yml
	Context("reconcile with postgres", func() {
		const postgresURL = "postgres://postgres:pA_sw0rd@localhost:5432?sslmode=disable"

		var (
			reconciler *DatabaseReconciler
			secret     *corev1.Secret
			database   *anbratengithubiov1alpha1.Database
		)

		reconcileDatabase := func() {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(database)})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(database), database)).To(Succeed())
		}

		connect := func(password string) error {
			config, err := pgx.ParseConfig(postgresURL)
			Expect(err).NotTo(HaveOccurred())
			config.User = "reconcile_user"
			config.Password = password
			config.Database = "reconcile_db"
			conn, err := pgx.ConnectConfig(ctx, config)
			if err != nil {
				return err
			}
			return conn.Close(ctx)
		}

		BeforeEach(func() {
			previousURL, hadURL := os.LookupEnv("POSTGRES_URL")
			Expect(os.Setenv("POSTGRES_URL", postgresURL)).To(Succeed())
			DeferCleanup(func() {
				if hadURL {
					Expect(os.Setenv("POSTGRES_URL", previousURL)).To(Succeed())
				} else {
					Expect(os.Unsetenv("POSTGRES_URL")).To(Succeed())
				}
			})

			reconciler = &DatabaseReconciler{Client: k8sClient, APIReader: k8sClient, Log: GinkgoLogr}

			secret = newSecret("reconcile-db", map[string]string{"password": `first'pass\1`})
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, secret)

			database = newDatabase("reconcile", "", ref("reconcile-db", "password"))
			database.Spec.Database = "reconcile_db"
			database.Spec.Username = "reconcile_user"
			Expect(k8sClient.Create(ctx, database)).To(Succeed())

			DeferCleanup(func() {
				// the finalizer removes user and database
				Expect(k8sClient.Delete(ctx, database)).To(Succeed())
				_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(database)})
				Expect(err).NotTo(HaveOccurred())
				Expect(errors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(database), database))).To(BeTrue())
				Expect(connect(`first'pass\1`)).NotTo(Succeed())
			})
		})

		It("creates the user with the password of the secret and updates it if the secret changes", func() {
			reconcileDatabase()
			Expect(connect(`first'pass\1`)).To(Succeed())
			Expect(database.Status.Created).To(BeTrue())
			Expect(database.Status.PasswordVersion).To(HavePrefix("secret/reconcile-db/password/"))

			// unchanged password: version stays the same
			version := database.Status.PasswordVersion
			reconcileDatabase()
			Expect(database.Status.PasswordVersion).To(Equal(version))

			// changed secret: password is updated
			secret.StringData = map[string]string{"password": `second'pass\2`}
			Expect(k8sClient.Update(ctx, secret)).To(Succeed())
			reconcileDatabase()
			Expect(database.Status.PasswordVersion).NotTo(Equal(version))
			Expect(connect(`second'pass\2`)).To(Succeed())
			Expect(connect(`first'pass\1`)).NotTo(Succeed())
		})

		It("updates the password when the secret changes while running in a manager", func() {
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:  scheme.Scheme,
				Metrics: metricsserver.Options{BindAddress: "0"},
				// envtest runs a single controller with this name per process
				Controller: config.Controller{SkipNameValidation: ptr.To(true)},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect((&DatabaseReconciler{
				Client:    mgr.GetClient(),
				APIReader: mgr.GetAPIReader(),
				Log:       GinkgoLogr,
				Scheme:    mgr.GetScheme(),
			}).SetupWithManager(mgr)).To(Succeed())

			mgrCtx, cancel := context.WithCancel(ctx)
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				Expect(mgr.Start(mgrCtx)).To(Succeed())
			}()
			DeferCleanup(func() {
				cancel()
				<-done
			})

			Eventually(func() error { return connect(`first'pass\1`) }, "10s").Should(Succeed())

			secret.StringData = map[string]string{"password": "changed-by-watch"}
			Expect(k8sClient.Update(ctx, secret)).To(Succeed())
			Eventually(func() error { return connect("changed-by-watch") }, "10s").Should(Succeed())
		})

		It("updates the password when switching from an inline password to a secret", func() {
			database.Spec.PasswordSecretRef = nil
			database.Spec.Password = "inline-pass"
			Expect(k8sClient.Update(ctx, database)).To(Succeed())
			reconcileDatabase()
			Expect(connect("inline-pass")).To(Succeed())
			Expect(database.Status.PasswordVersion).To(HavePrefix("spec/"))

			database.Spec.Password = ""
			database.Spec.PasswordSecretRef = ref("reconcile-db", "password")
			Expect(k8sClient.Update(ctx, database)).To(Succeed())
			reconcileDatabase()
			Expect(connect(`first'pass\1`)).To(Succeed())
			Expect(connect("inline-pass")).NotTo(Succeed())
		})
	})
})
