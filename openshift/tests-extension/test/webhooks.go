package test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

const (
	webhookCatalogName         = "webhook-operator-catalog"
	webhookOperatorPackageName = "webhook-operator"
	webhookOperatorCRDName     = "webhooktests.webhook.operators.coreos.io"
	webhookServiceCert         = "webhook-operator-webhook-service-cert"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMWebhookProviderOpenshiftServiceCA][Skipped:Disconnected][Serial] OLMv1 operator with webhooks",
	Ordered, Serial, func() {
		var (
			k8sClient                       client.Client
			dynamicClient                   dynamic.Interface
			webhookOperatorInstallNamespace string
			cleanup                         func(ctx context.Context)
		)

		BeforeEach(func(ctx SpecContext) {
			By("initializing Kubernetes client")
			k8sClient = env.Get().K8sClient
			restCfg := env.Get().RestCfg
			var err error
			dynamicClient, err = dynamic.NewForConfig(restCfg)
			Expect(err).ToNot(HaveOccurred(), "failed to create dynamic client")

			By("requiring OLMv1 capability on OpenShift")
			helpers.RequireOLMv1CapabilityOnOpenshift()

			By("ensuring no ClusterExtension and CRD from a previous run")
			helpers.EnsureCleanupClusterExtension(ctx, webhookOperatorPackageName, webhookOperatorCRDName)

			By(fmt.Sprintf("checking if the %s exists", webhookCatalogName))
			catalog := &olmv1.ClusterCatalog{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: webhookCatalogName}, catalog)
			if apierrors.IsNotFound(err) {
				By(fmt.Sprintf("creating the webhook-operator catalog with name %s", webhookCatalogName))
				catalog = helpers.NewClusterCatalog(webhookCatalogName, "quay.io/operator-framework/webhook-operator-index:0.0.3")
				err = k8sClient.Create(ctx, catalog)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the webhook-operator catalog to be serving")
				helpers.ExpectCatalogToBeServing(ctx, webhookCatalogName)
			} else {
				By(fmt.Sprintf("webhook-operator catalog %s already exists, skipping creation", webhookCatalogName))
			}
			webhookOperatorInstallNamespace = fmt.Sprintf("webhook-operator-%s", rand.String(5))
			cleanup = setupWebhookOperator(ctx, k8sClient, webhookOperatorInstallNamespace)
		})

		AfterEach(func(ctx SpecContext) {
			if CurrentSpecReport().Failed() {
				By("dumping pod logs for debugging")
				helpers.GetAllPodLogs(ctx, webhookOperatorInstallNamespace)
				helpers.DescribePods(ctx, webhookOperatorInstallNamespace)
				helpers.DescribeAllClusterCatalogs(ctx)
				helpers.DescribeAllClusterExtensions(ctx, webhookOperatorInstallNamespace)
				By("dumping webhook diagnostics")
				// Additional diagnostics specific for this test
				helpers.RunAndPrint(ctx, "get", "mutatingwebhookconfigurations.admissionregistration.k8s.io", "-oyaml")
				helpers.RunAndPrint(ctx, "get", "validatingwebhookconfigurations.admissionregistration.k8s.io", "-oyaml")
			}

			By("performing webhook operator cleanup")
			if cleanup != nil {
				cleanup(ctx)
			}
		})

		AfterAll(func(ctx SpecContext) {
			helpers.EnsureCleanupClusterCatalog(ctx, webhookCatalogName)
		})

		It("should have a working validating webhook", func(ctx SpecContext) {
			By("creating a webhook test resource that will be rejected by the validating webhook")
			Eventually(func() error {
				name := fmt.Sprintf("validating-webhook-test-%s", rand.String(5))
				obj := newWebhookTest(name, webhookOperatorInstallNamespace, false)

				_, err := dynamicClient.
					Resource(webhookTestV1).
					Namespace(webhookOperatorInstallNamespace).
					Create(ctx, obj, metav1.CreateOptions{})

				switch {
				case err == nil:
					// Webhook not ready yet; clean up and keep polling.
					_ = dynamicClient.Resource(webhookTestV1).
						Namespace(webhookOperatorInstallNamespace).
						Delete(ctx, name, metav1.DeleteOptions{})
					return fmt.Errorf("webhook not rejecting yet")
				case strings.Contains(err.Error(), "Invalid value: false: Spec.Valid must be true"):
					return nil // got the expected validating-webhook rejection
				default:
					return fmt.Errorf("unexpected error: %v", err)
				}
			}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		})

		It("should have a working mutating webhook", func(ctx SpecContext) {
			By("creating a valid webhook")
			mutatingWebhookResourceName := "mutating-webhook-test"
			resource := newWebhookTest(mutatingWebhookResourceName, webhookOperatorInstallNamespace, true)
			Eventually(func(g Gomega) {
				_, err := dynamicClient.Resource(webhookTestV1).Namespace(webhookOperatorInstallNamespace).Create(ctx, resource, metav1.CreateOptions{})
				g.Expect(err).ToNot(HaveOccurred())
			}).WithTimeout(1 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("getting the created resource in v1 schema")
			obj, err := dynamicClient.Resource(webhookTestV1).Namespace(webhookOperatorInstallNamespace).Get(ctx, mutatingWebhookResourceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(BeNil())

			By("validating the resource spec")
			spec := obj.Object["spec"].(map[string]interface{})
			Expect(spec).To(Equal(map[string]interface{}{
				"valid":  true,
				"mutate": true,
			}))
		})

		It("should have a working conversion webhook", func(ctx SpecContext) {
			By("creating a conversion webhook test resource")
			conversionWebhookResourceName := "conversion-webhook-test"
			resourceV1 := newWebhookTest(conversionWebhookResourceName, webhookOperatorInstallNamespace, true)
			Eventually(func(g Gomega) {
				_, err := dynamicClient.Resource(webhookTestV1).Namespace(webhookOperatorInstallNamespace).Create(ctx, resourceV1, metav1.CreateOptions{})
				g.Expect(err).ToNot(HaveOccurred())
			}).WithTimeout(1 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("getting the created resource in v2 schema")
			obj, err := dynamicClient.Resource(webhookTestV2).Namespace(webhookOperatorInstallNamespace).Get(ctx, conversionWebhookResourceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(BeNil())

			By("validating the resource spec")
			spec := obj.Object["spec"].(map[string]interface{})
			Expect(spec).To(Equal(map[string]interface{}{
				"conversion": map[string]interface{}{
					"valid":  true,
					"mutate": true,
				},
			}))
		})

		It("should be tolerant to tls secret deletion", func(ctx SpecContext) {
			certificateSecretName := webhookServiceCert
			By("ensuring secret exists before deletion attempt")
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, client.ObjectKey{Name: certificateSecretName, Namespace: webhookOperatorInstallNamespace}, secret)
				g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get secret %s/%s", webhookOperatorInstallNamespace, certificateSecretName))
			}).WithTimeout(1 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("checking webhook is responsive through secret recreation after manual deletion")
			tlsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      certificateSecretName,
					Namespace: webhookOperatorInstallNamespace,
				},
			}
			err := k8sClient.Delete(ctx, tlsSecret, client.PropagationPolicy(metav1.DeletePropagationBackground))
			Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())

			DeferCleanup(func() {
				// Specific check for this test
				if CurrentSpecReport().Failed() {
					By("dumping certificate details for debugging")
					secret := &corev1.Secret{}
					if err := k8sClient.Get(ctx, client.ObjectKey{
						Name:      webhookServiceCert,
						Namespace: webhookOperatorInstallNamespace,
					}, secret); err == nil {
						if crt, ok := secret.Data["tls.crt"]; ok && len(crt) > 0 {
							printTLSCertInfo(crt)
						} else {
							_, _ = GinkgoWriter.Write([]byte("[diag] tls.crt key not found or empty in secret\n"))
						}
					} else {
						fmt.Fprintf(GinkgoWriter, "[diag] failed to get secret for cert dump: %v\n", err)
					}
				}
			})

			By("waiting for the webhook operator's service certificate secret to be recreated and populated")
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, client.ObjectKey{Name: certificateSecretName, Namespace: webhookOperatorInstallNamespace}, secret)
				if apierrors.IsNotFound(err) {
					GinkgoLogr.Info(fmt.Sprintf("Secret %s/%s not found yet (still polling for recreation)", webhookOperatorInstallNamespace, certificateSecretName))
					return
				}
				g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get webhook service certificate secret %s/%s: %v", webhookOperatorInstallNamespace, certificateSecretName, err))
				g.Expect(secret.Data).ToNot(BeEmpty(), "expected webhook service certificate secret data to not be empty after recreation")
			}).WithTimeout(5*time.Minute).WithPolling(10*time.Second).Should(Succeed(), "webhook service certificate secret did not get recreated and populated within timeout")

			Eventually(func(g Gomega) {
				resourceName := fmt.Sprintf("tls-deletion-test-%s", rand.String(5))
				resource := newWebhookTest(resourceName, webhookOperatorInstallNamespace, true)

				_, err := dynamicClient.Resource(webhookTestV1).Namespace(webhookOperatorInstallNamespace).Create(ctx, resource, metav1.CreateOptions{})
				g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create test resource %s: %v", resourceName, err))

				err = dynamicClient.Resource(webhookTestV1).Namespace(webhookOperatorInstallNamespace).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
				g.Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred(), fmt.Sprintf("failed to delete test resource %s: %v", resourceName, err))
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
		})
	})

var webhookTestV1 = schema.GroupVersionResource{
	Group:    "webhook.operators.coreos.io",
	Version:  "v1",
	Resource: "webhooktests",
}

var webhookTestV2 = schema.GroupVersionResource{
	Group:    "webhook.operators.coreos.io",
	Version:  "v2",
	Resource: "webhooktests",
}

func newWebhookTest(name, namespace string, valid bool) *unstructured.Unstructured {
	mutateValue := valid
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "webhook.operators.coreos.io/v1",
			"kind":       "WebhookTest",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"valid":  valid,
				"mutate": mutateValue,
			},
		},
	}
	return obj
}

func setupWebhookOperator(ctx SpecContext, k8sClient client.Client, webhookOperatorInstallNamespace string) func(ctx context.Context) {
	By(fmt.Sprintf("installing the webhook operator in namespace %s", webhookOperatorInstallNamespace))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: webhookOperatorInstallNamespace},
	}
	err := k8sClient.Create(ctx, ns)
	Expect(err).ToNot(HaveOccurred())

	saName := fmt.Sprintf("%s-installer", webhookOperatorInstallNamespace)
	sa := helpers.NewServiceAccount(saName, webhookOperatorInstallNamespace)
	err = k8sClient.Create(ctx, sa)
	Expect(err).ToNot(HaveOccurred())
	helpers.ExpectServiceAccountExists(ctx, saName, webhookOperatorInstallNamespace)

	By("creating a ClusterRoleBinding to cluster-admin for the webhook operator")
	operatorClusterRoleBindingName := fmt.Sprintf("%s-operator-crb", webhookOperatorInstallNamespace)
	operatorClusterRoleBinding := helpers.NewClusterRoleBinding(operatorClusterRoleBindingName, "cluster-admin", saName, webhookOperatorInstallNamespace)
	err = k8sClient.Create(ctx, operatorClusterRoleBinding)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create ClusterRoleBinding %s",
		operatorClusterRoleBindingName))
	helpers.ExpectClusterRoleBindingExists(ctx, operatorClusterRoleBindingName)

	ceName := webhookOperatorInstallNamespace
	ce := helpers.NewClusterExtensionObject("webhook-operator", "0.0.1", ceName, saName, webhookOperatorInstallNamespace)
	ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"olm.operatorframework.io/metadata.name": webhookCatalogName,
		},
	}
	err = k8sClient.Create(ctx, ce)
	Expect(err).ToNot(HaveOccurred())

	By("waiting for the webhook operator to be installed")
	helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)

	By("waiting for the webhook operator's service to be ready")
	serviceName := "webhook-operator-webhook-service" // Standard name for the service created by the operator
	Eventually(func(g Gomega) {
		svc := &corev1.Service{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: webhookOperatorInstallNamespace}, svc)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get webhook service %s/%s: %v", webhookOperatorInstallNamespace, serviceName, err))
		g.Expect(svc.Spec.ClusterIP).ToNot(BeEmpty(), "expected webhook service to have a ClusterIP assigned")
		g.Expect(svc.Spec.Ports).ToNot(BeEmpty(), "expected webhook service to have ports defined")
	}).WithTimeout(1*time.Minute).WithPolling(5*time.Second).Should(Succeed(), "webhook service did not become ready within timeout")

	By("waiting for the webhook operator's service certificate secret to exist and be populated")
	certificateSecretName := "webhook-operator-webhook-service-cert" // Fixed to use the static name
	Eventually(func(g Gomega) {
		secret := &corev1.Secret{}
		// Force bypassing the client cache for this Get operation
		err := k8sClient.Get(ctx, client.ObjectKey{Name: certificateSecretName, Namespace: webhookOperatorInstallNamespace}, secret) // Removed client.WithCacheDisabled

		if apierrors.IsNotFound(err) {
			GinkgoLogr.Info(fmt.Sprintf("Secret %s/%s not found yet (still polling)", webhookOperatorInstallNamespace, certificateSecretName))
			return // Keep polling if not found
		}

		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get webhook service certificate secret %s/%s: %v",
			webhookOperatorInstallNamespace, certificateSecretName, err))
		g.Expect(secret.Data).ToNot(BeEmpty(), "expected webhook service certificate secret data to not be empty")
	}).WithTimeout(5*time.Minute).WithPolling(5*time.Second).Should(Succeed(), "webhook service certificate secret did not become available within timeout")

	return func(ctx context.Context) {
		By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
		_ = k8sClient.Delete(ctx, ce, client.PropagationPolicy(metav1.DeletePropagationBackground))
		By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", operatorClusterRoleBinding.Name))
		_ = k8sClient.Delete(ctx, operatorClusterRoleBinding, client.PropagationPolicy(metav1.DeletePropagationBackground))
		By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
		_ = k8sClient.Delete(ctx, sa, client.PropagationPolicy(metav1.DeletePropagationBackground))
		By(fmt.Sprintf("cleanup: deleting namespace %s", ns.Name))
		_ = k8sClient.Delete(ctx, ns, client.PropagationPolicy(metav1.DeletePropagationForeground))

		By(fmt.Sprintf("waiting for namespace %s to be fully deleted", webhookOperatorInstallNamespace))
		Eventually(func(g Gomega) {
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: webhookOperatorInstallNamespace}, ns)
			g.Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "namespace still exists")
		}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
	}
}

// printTLSCertInfo parses a PEM-encoded TLS certificate and prints useful debug info.
// It shows validity period and SANs (DNS/IP) to help debug webhook cert issues.
func printTLSCertInfo(certPEM []byte) {
	fmt.Fprintln(GinkgoWriter, "\n[diag] === TLS Certificate Info ===")

	block, _ := pem.Decode(certPEM)
	if block == nil {
		fmt.Fprintln(GinkgoWriter, "[diag] failed to decode PEM block from tls.crt")
		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "[diag] failed to parse certificate: %v\n", err)
		return
	}

	fmt.Fprintf(GinkgoWriter, "[diag] Subject: %s\n", cert.Subject.String())
	fmt.Fprintf(GinkgoWriter, "[diag] Issuer: %s\n", cert.Issuer.String())
	fmt.Fprintf(GinkgoWriter, "[diag] Serial Number: %X\n", cert.SerialNumber)
	fmt.Fprintf(GinkgoWriter, "[diag] Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Fprintf(GinkgoWriter, "[diag] Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))
	fmt.Fprintf(GinkgoWriter, "[diag] IsCA: %t\n", cert.IsCA)

	sans := make([]string, 0, len(cert.DNSNames)+len(cert.IPAddresses))
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}
	if len(sans) > 0 {
		fmt.Fprintf(GinkgoWriter, "[diag] SANs: %s\n", strings.Join(sans, ", "))
	} else {
		fmt.Fprintln(GinkgoWriter, "[diag] SANs: <none>")
	}

	fmt.Fprintln(GinkgoWriter, "[diag] ===========================")
}
