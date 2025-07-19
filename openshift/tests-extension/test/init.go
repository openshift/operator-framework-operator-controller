package test

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
)

var (
	// RestCfg is the unified REST config used by all clients
	RestCfg *rest.Config

	// K8sClient is the controller-runtime client
	K8sClient crclient.Client

	// IsOpenShift indicates whether the cluster is an OpenShift cluster
	IsOpenShift bool
)

func init() {
	var err error

	// Step 1: Build the REST config (admin if possible)
	RestCfg = getRestConfig()

	// Step 2: Use discovery to detect OpenShift
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(RestCfg)
	IsOpenShift = detectOpenShift(discoveryClient)

	// Step 3: Build the runtime scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	if IsOpenShift {
		utilruntime.Must(configv1.AddToScheme(scheme))
	}

	// Step 4: Initialize controller-runtime client
	K8sClient, err = crclient.New(RestCfg, crclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("failed to create controller-runtime client: %v", err)
	}
}

// getRestConfig returns a high-QPS *rest.Config, preferring admin credentials
func getRestConfig() *rest.Config {
	paths := []string{
		"/etc/origin/admin.kubeconfig",
		"/etc/kubernetes/kubeconfig",
		os.Getenv("KUBECONFIG"),
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			log.Printf("Using kubeconfig: %s", path)
			cfg, err := clientcmd.BuildConfigFromFlags("", path)
			if err != nil {
				log.Fatalf("Failed to load kubeconfig from %s: %v", path, err)
			}
			return turnOffRateLimiting(cfg)
		}
	}

	log.Printf("Falling back to in-cluster config")
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to load in-cluster config: %v", err)
	}
	return turnOffRateLimiting(cfg)
}

// detectOpenShift returns true if OpenShift APIs are detected
func detectOpenShift(d discovery.DiscoveryInterface) bool {
	groups, err := d.ServerGroups()
	if err != nil {
		log.Printf("Warning: failed to list API groups: %v", err)
		return false
	}
	for _, g := range groups.Groups {
		if g.Name == "config.openshift.io" {
			log.Printf("Detected OpenShift cluster")
			return true
		}
	}
	log.Printf("Detected Kubernetes cluster")
	return false
}

// turnOffRateLimiting configures aggressive QPS/burst settings
func turnOffRateLimiting(cfg *rest.Config) *rest.Config {
	cfg.QPS = 10000
	cfg.Burst = 10000
	cfg.RateLimiter = nil
	return cfg
}
