package env

import (
	"log"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github/operator-framework-operator-controller/openshift/tests-extension/test/extlogs"
)

type TestEnv struct {
	RestCfg     *rest.Config
	K8sClient   crclient.Client
	IsOpenShift bool
}

// Global shared instance
var testEnv *TestEnv

func Get() *TestEnv {
	if testEnv == nil {
		log.Fatalf("env.TestEnv was not initialized — call Init() first")
	}
	return testEnv
}

// Init initializes the test environment, setting up the REST config and client.
func Init() *TestEnv {
	if testEnv == nil {
		testEnv = initTestEnv()
	}
	return testEnv
}

// initTestEnv initializes the test environment, setting up the REST config,
func initTestEnv() *TestEnv {
	cfg := getRestConfig()

	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	isOcp := detectOpenShift(discoveryClient)

	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	if isOcp {
		utilruntime.Must(configv1.AddToScheme(scheme))
	}

	k8sClient, err := crclient.New(cfg, crclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("failed to create controller-runtime client: %v", err)
	}

	if isOcp {
		extlogs.Infof("[env] Cluster environment initialized (OpenShift: %t)\n", isOcp)
	}

	return &TestEnv{
		RestCfg:     cfg,
		K8sClient:   k8sClient,
		IsOpenShift: isOcp,
	}
}

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

func configureQPS(cfg *rest.Config) *rest.Config {
	cfg.QPS = 10000
	cfg.Burst = 10000
	cfg.RateLimiter = nil
	return cfg
}
