package main

import (
	"flag"
	"os"
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

	capabilities, err := getCapabilities(kubeClient)
	if err != nil {
		klog.Fatalf("Error retrieving Kubernetes cluster capabilities: %s", err.Error())
	}

	var ctrl controller
	if capabilities.Has("extensions/v1beta1") {
		ctrl = controllerv1beta1.NewController(
			kubeClient,
			faasClient,
			kubeInformerFactory,
			faasInformerFactory,
		)
	} else {
		ctrl = controllerv1.NewController(
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
