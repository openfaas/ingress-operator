package main

import (
	"flag"
	"os"
	"strings"
	"time"

	clientset "github.com/openfaas-incubator/ingress-operator/pkg/client/clientset/versioned"
	informers "github.com/openfaas-incubator/ingress-operator/pkg/client/informers/externalversions"
	controllerv1 "github.com/openfaas-incubator/ingress-operator/pkg/controller/v1"
	controllerv1beta1 "github.com/openfaas-incubator/ingress-operator/pkg/controller/v1beta1"
	"github.com/openfaas-incubator/ingress-operator/pkg/signals"
	"github.com/openfaas-incubator/ingress-operator/pkg/version"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog"

	// required for generating code from CRD
	_ "k8s.io/code-generator/cmd/client-gen/generators"

	// required to authenticate against GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	masterURL  string
	kubeconfig string
)

var pullPolicyOptions = map[string]bool{
	"Always":       true,
	"IfNotPresent": true,
	"Never":        true,
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	// TODO: remove
	flag.Bool("logtostderr", false, "logtostderr legacy flag")
}

func main() {
	// TODO: remove
	flag.Set("logtostderr", "true")
	flag.Parse()

	setupLogging()

	sha, release := version.GetReleaseInfo()
	klog.Infof("Starting FunctionIngress controller version: %s commit: %s", release, sha)

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building Kubernetes clientset: %s", err.Error())
	}

	faasClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building FunctionIngress clientset: %s", err.Error())
	}

	ingressNamespace := "openfaas"
	if namespace, exists := os.LookupEnv("ingress_namespace"); exists {
		ingressNamespace = namespace
	}

	defaultResync := time.Second * 30

	kubeInformerOpt := kubeinformers.WithNamespace(ingressNamespace)
	kubeInformerFactory := kubeinformers.
		NewSharedInformerFactoryWithOptions(kubeClient, defaultResync, kubeInformerOpt)

	faasInformerOpt := informers.WithNamespace(ingressNamespace)
	faasInformerFactory := informers.
		NewSharedInformerFactoryWithOptions(faasClient, defaultResync, faasInformerOpt)

	capabilities, err := getPreferredAvailableAPIs(kubeClient, "Ingress")
	if err != nil {
		klog.Fatalf("Error retrieving Kubernetes cluster capabilities: %s", err.Error())
	}

	klog.Infof("cluster supports ingress in: %s", capabilities)

	var ctrl controller
	// prefer v1, if it is available, this removes any deprecation warnings
	if capabilities.Has("networking.k8s.io/v1") {
		ctrl = controllerv1.NewController(
			kubeClient,
			faasClient,
			kubeInformerFactory,
			faasInformerFactory,
		)
	} else {
		// use v1beta1 by default
		ctrl = controllerv1beta1.NewController(
			kubeClient,
			faasClient,
			kubeInformerFactory,
			faasInformerFactory,
		)
	}

	go kubeInformerFactory.Start(stopCh)
	go faasInformerFactory.Start(stopCh)

	if err = ctrl.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

type controller interface {
	Run(int, <-chan struct{}) error
}

func setupLogging() {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// Sync the klog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})
}

type Capabilities map[string]bool

func (c Capabilities) Has(wanted string) bool {
	return c[wanted]
}

func (c Capabilities) String() string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// getCapabilities returns the list of available api groups in the cluster.
func getCapabilities(client kubernetes.Interface) (Capabilities, error) {

	groupList, err := client.Discovery().ServerGroups()
	if err != nil {
		return nil, err
	}

	caps := Capabilities{}
	for _, g := range groupList.Groups {
		for _, gv := range g.Versions {
			caps[gv.GroupVersion] = true
		}
	}

	return caps, nil
}

// getPreferredAvailableAPIs queries the cluster for the preferred resources information and returns a Capabilities
// instance containing those api groups that support the specified kind.
//
// kind should be the title case singular name of the kind. For example, "Ingress" is the kind for a resource "ingress".
func getPreferredAvailableAPIs(client kubernetes.Interface, kind string) (Capabilities, error) {
	discoveryclient := client.Discovery()
	lists, err := discoveryclient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	caps := Capabilities{}
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}
			if resource.Kind == kind {
				caps[list.GroupVersion] = true
			}
		}
	}

	return caps, nil
}
