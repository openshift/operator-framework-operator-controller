package env

import (
	"context"
	"fmt"
	"log"
	"os"

	bsemver "github.com/blang/semver/v4"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/extlogs"
)

// TestEnv holds the test environment state, including the Kubernetes REST config,
// controller-runtime client, and a flag indicating if the cluster is OpenShift.
type TestEnv struct {
	// RestCfg stores the Kubernetes REST configuration used by clients
	RestCfg *rest.Config

	// Controller-runtime client for interacting with the cluster
	K8sClient crclient.Client

	// True if the cluster is detected as an OpenShift environment
	IsOpenShift bool

	// Set to the MAJOR.MINOR version of OpenShift, blank otherwise
	OpenShiftVersion string
}

// testEnv is the global shared instance used by all tests.
// It must be initialized via Init() before use.
var testEnv *TestEnv

// Get returns the initialized test environment.
// It will panic if Init() has not been called first.
func Get() *TestEnv {
	if testEnv == nil {
		log.Fatalf("env.TestEnv was not initialized — call Init() first")
	}
	return testEnv
}

// Init sets up the global test environment if it hasn't been initialized yet.
// It creates the REST config, client, and cluster metadata used by tests.
func Init() *TestEnv {
	if testEnv == nil {
		testEnv = initTestEnv()
	}
	return testEnv
}

// initTestEnv initializes the test environment by setting up the Kubernetes REST config,
// discovering whether the cluster is OpenShift, registering required API schemes,
// and creating a controller-runtime client. This is used to build the shared TestEnv object
// that provides access to the API and client in tests.
// You should call this function before any suite that requires access to the cluster:
// Example usage:
//
//		BeforeEach(func() {
//	    _ = env.Init()
//		})
func initTestEnv() *TestEnv {
	cfg := getRestConfig()
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	isOcp := detectOpenShift(discoveryClient)

	// Create the runtime scheme and register all necessary types
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
	utilruntime.Must(olmv1.AddToScheme(scheme))

	if isOcp {
		utilruntime.Must(buildv1.AddToScheme(scheme))
		utilruntime.Must(configv1.AddToScheme(scheme))
		utilruntime.Must(imagev1.AddToScheme(scheme))
		utilruntime.Must(operatorv1.AddToScheme(scheme))
	}

	k8sClient, err := crclient.New(cfg, crclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("failed to create controller-runtime client: %v", err)
	}

	version := ""
	if isOcp {
		extlogs.Infof("[env] Cluster environment initialized (OpenShift: %t)\n", isOcp)
		version = getOcpVersion(k8sClient)
	}

	return &TestEnv{
		RestCfg:          cfg,
		K8sClient:        k8sClient,
		IsOpenShift:      isOcp,
		OpenShiftVersion: version,
	}
}

func getOcpVersion(c crclient.Client) string {
	cv := &configv1.ClusterVersion{}
	err := c.Get(context.Background(), crclient.ObjectKey{Name: "version"}, cv)
	if err != nil {
		return ""
	}
	v, err := bsemver.Parse(cv.Status.Desired.Version)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// getRestConfig returns a Kubernetes REST config for the test client.
// It first checks the KUBECONFIG environment variable and uses that if available.
// If not, it falls back to using in-cluster configuration (when running inside a pod).
// This allows the same test code to run in both local and cluster environments.
func getRestConfig() *rest.Config {
	kubeconfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(kubeconfig); err == nil {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Failed to load kubeconfig from %s: %v", kubeconfig, err)
		}
		extlogs.Infof("[env] Using kubeconfig: %s\n", kubeconfig)
		return configureQPS(cfg)
	}
	extlogs.Infof("[env] Using in-cluster configuration: %s\n", kubeconfig)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to load in-cluster config: %v", err)
	}
	return configureQPS(cfg)
}

// detectOpenShift checks if the cluster is an OpenShift cluster.
// It does this by looking for the "config.openshift.io" API group,
// which only exists in OpenShift environments.
func detectOpenShift(d discovery.DiscoveryInterface) bool {
	groups, err := d.ServerGroups()
	if err != nil {
		extlogs.WarnContextf("failed to list API groups: %v", err)
		return false
	}
	for _, g := range groups.Groups {
		if g.Name == "config.openshift.io" {
			return true
		}
	}
	return false
}

// configureQPS sets high QPS and burst values to avoid client-side throttling during tests.
// This makes tests faster by allowing many API calls without delay.
// It's mainly needed in serial tests, where slow or throttled requests can cause flakes.
// The default limits (QPS=5, Burst=10) are too low for most test workloads.
func configureQPS(cfg *rest.Config) *rest.Config {
	cfg.QPS = 10000
	cfg.Burst = 10000
	cfg.RateLimiter = nil
	return cfg
}
