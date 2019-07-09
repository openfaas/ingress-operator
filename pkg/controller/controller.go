package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/google/go-cmp/cmp"
	faasv1 "github.com/openfaas-incubator/ingress-operator/pkg/apis/openfaas/v1alpha2"
	clientset "github.com/openfaas-incubator/ingress-operator/pkg/client/clientset/versioned"
	faasscheme "github.com/openfaas-incubator/ingress-operator/pkg/client/clientset/versioned/scheme"
	informers "github.com/openfaas-incubator/ingress-operator/pkg/client/informers/externalversions"
	listers "github.com/openfaas-incubator/ingress-operator/pkg/client/listers/openfaas/v1alpha2"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	glog "k8s.io/klog"
)

const controllerAgentName = "ingress-operator"
const faasIngressKind = "FunctionIngress"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Function is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Function fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by controller"
	// MessageResourceSynced is the message used for an Event fired when a Function
	// is synced successfully
	MessageResourceSynced = "FunctionIngress synced successfully"
)

// Controller is the controller implementation for Function resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// faasclientset is a clientset for our own API group
	faasclientset clientset.Interface

	functionsLister listers.FunctionIngressLister

	functionsSynced cache.InformerSynced

	ingressLister extensionsv1beta1.IngressLister

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func checkCustomResourceType(obj interface{}) (faasv1.FunctionIngress, bool) {
	var fn *faasv1.FunctionIngress
	var ok bool
	if fn, ok = obj.(*faasv1.FunctionIngress); !ok {
		glog.Errorf("Event Watch received an invalid object: %#v", obj)
		return faasv1.FunctionIngress{}, false
	}
	return *fn, true
}

// NewController returns a new OpenFaaS controller
func NewController(
	kubeclientset kubernetes.Interface,
	faasclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	functionIngressFactory informers.SharedInformerFactory) *Controller {

	functionIngress := functionIngressFactory.Openfaas().V1alpha2().FunctionIngresses()

	// Create event broadcaster
	// Add o6s types to the default Kubernetes Scheme so Events can be
	// logged for faas-controller types.
	faasscheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.V(4).Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	ingressInformer := kubeInformerFactory.Extensions().V1beta1().Ingresses()
	ingressLister := ingressInformer.Lister()

	controller := &Controller{
		kubeclientset:   kubeclientset,
		faasclientset:   faasclientset,
		functionsLister: functionIngress.Lister(),
		functionsSynced: functionIngress.Informer().HasSynced,
		ingressLister:   ingressLister,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "FunctionIngresses"),
		recorder:        recorder,
	}

	glog.Info("Setting up event handlers")

	//  Add FunctionIngress Informer
	//
	// Set up an event handler for when FunctionIngress resources change
	functionIngress.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueFunction,
		UpdateFunc: func(old, new interface{}) {
			oldFn, ok := checkCustomResourceType(old)
			if !ok {
				return
			}
			newFn, ok := checkCustomResourceType(new)
			if !ok {
				return
			}
			if diff := cmp.Diff(oldFn.Spec, newFn.Spec); diff != "" {
				controller.enqueueFunction(new)
			}
		},
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: controller.handleObject,
	})

	// Set up an event handler for when functions related resources like pods, deployments, replica sets
	// can't be materialized. This logs abnormal events like ImagePullBackOff, back-off restarting failed container,
	// failed to start container, oci runtime errors, etc
	// Enable this with -v=3
	kubeInformerFactory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				event := obj.(*corev1.Event)
				since := time.Since(event.LastTimestamp.Time)
				// log abnormal events occurred in the last minute
				if since.Seconds() < 61 && strings.Contains(event.Type, "Warning") {
					glog.V(3).Infof("Abnormal event detected on %s %s: %s", event.LastTimestamp, key, event.Message)
				}
			}
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.functionsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Function resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		c.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Function resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Function resource with this namespace/name
	function, err := c.functionsLister.FunctionIngresses(namespace).Get(name)
	if err != nil {
		// The Function resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("function ingress '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	fniName := function.ObjectMeta.Name
	glog.Infof("FunctionIngress name: %v", fniName)

	ingresses := c.ingressLister.Ingresses(namespace)
	ingress, getIngressErr := ingresses.Get(function.Name)
	createIngress := errors.IsNotFound(getIngressErr)
	if !createIngress && ingress == nil {
		glog.Errorf("cannot get ingress: %s in %s, error: %s", function.Name, namespace, getIngressErr.Error())
	}

	glog.Info("function.Spec.UseTLS() ", function.Spec.UseTLS())
	glog.Info("createIngress ", createIngress)

	if createIngress {
		rules := makeRules(function)
		tls := makeTLS(function)

		newIngress := v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       namespace,
				Annotations:     makeAnnotations(function),
				OwnerReferences: makeOwnerRef(function),
			},
			Spec: v1beta1.IngressSpec{
				Rules: rules,
				TLS:   tls,
			},
		}

		_, createErr := c.kubeclientset.Extensions().Ingresses(namespace).Create(&newIngress)
		if createErr != nil {
			glog.Errorf("cannot create ingress: %v in %v, error: %v", name, namespace, createErr.Error())
		}

		c.recorder.Event(function, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
		return nil
	}

	old := faasv1.FunctionIngress{}

	if val, ok := ingress.Annotations["com.openfaas.spec"]; ok && len(val) > 0 {
		unmarshalErr := json.Unmarshal([]byte(val), &old)
		if unmarshalErr != nil {
			return pkgerrors.Wrap(unmarshalErr, "unable to unmarshal from field com.openfaas.spec")
		}
	}

	// Update the Deployment resource if the Function definition differs
	if ingressNeedsUpdate(&old, function) {
		glog.Infof("Need to update FunctionIngress: %v", fniName)

		if old.ObjectMeta.Name != function.ObjectMeta.Name {
			return fmt.Errorf("cannot rename object")
		}

		updated := ingress.DeepCopy()

		rules := makeRules(function)

		annotations := makeAnnotations(function)
		for k, v := range annotations {
			updated.Annotations[k] = v
		}

		updated.Spec.Rules = rules
		updated.Spec.TLS = makeTLS(function)

		_, updateErr := c.kubeclientset.Extensions().Ingresses(namespace).Update(updated)
		if updateErr != nil {
			glog.Errorf("error updating ingress: %v", updateErr)
			return updateErr
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return fmt.Errorf("transient error: %v", err)
	}

	c.recorder.Event(function, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func ingressNeedsUpdate(old, function *faasv1.FunctionIngress) bool {

	return !cmp.Equal(old.Spec, function.Spec)
}

func (c *Controller) updateFunctionStatus(function *faasv1.FunctionIngress, deployment *appsv1beta2.Deployment) error {
	// TODO: enable status on K8s 1.12
	return nil
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	functionCopy := function.DeepCopy()
	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the Function resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	_, err := c.faasclientset.OpenfaasV1alpha2().FunctionIngresses(function.Namespace).Update(functionCopy)
	return err
}

// enqueueFunction takes a Function resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Function.
func (c *Controller) enqueueFunction(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Function resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Function resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a function, we should not do anything more
		// with it.
		if ownerRef.Kind != faasIngressKind {
			return
		}

		function, err := c.functionsLister.FunctionIngresses(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.Infof("FunctionIngress '%s' deleted. Ignoring orphaned object '%s'", ownerRef.Name, object.GetSelfLink())
			return
		}

		c.enqueueFunction(function)
		return
	}
}

func makeRules(fni *faasv1.FunctionIngress) []v1beta1.IngressRule {
	path := "/(.*)"
	if getClass(fni.Spec.IngressType) == "traefik" {
		path = "/"
	}

	return []v1beta1.IngressRule{
		v1beta1.IngressRule{
			Host: fni.Spec.Domain,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						v1beta1.HTTPIngressPath{
							Path: path,
							Backend: v1beta1.IngressBackend{
								ServiceName: "gateway",
								ServicePort: intstr.IntOrString{
									IntVal: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
}

func makeTLS(function *faasv1.FunctionIngress) []v1beta1.IngressTLS {
	if !function.Spec.UseTLS() {
		return []v1beta1.IngressTLS{}
	}
	return []v1beta1.IngressTLS{
		v1beta1.IngressTLS{
			SecretName: function.ObjectMeta.Name + "-cert",
			Hosts: []string{
				function.Spec.Domain,
			},
		},
	}
}

func getClass(ingressType string) string {
	switch ingressType {
	case "":
	case "nginx":
		return "nginx"
	default:
		return ingressType
	}

	return "nginx"
}

func makeAnnotations(function *faasv1.FunctionIngress) map[string]string {
	class := getClass(function.Spec.IngressType)
	specJSON, _ := json.Marshal(function)
	annotations := map[string]string{
		"kubernetes.io/ingress.class": class,
		"com.openfaas.spec":           string(specJSON),
	}

	switch class {

	case "nginx":
		annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/function/" + function.Spec.Function + "/$1"
		break
	case "skipper":
		annotations["zalando.org/skipper-filter"] = `setPath("/function/` + function.Spec.Function + `")`
		break

	case "traefik":
		annotations["traefik.ingress.kubernetes.io/rewrite-target"] = "/function/" + function.Spec.Function + "/$1"
		annotations["traefik.ingress.kubernetes.io/rule-type"] = `PathPrefix`

		break
	}

	if function.Spec.UseTLS() {
		annotations["certmanager.k8s.io/issuer"] = function.Spec.TLS.IssuerRef.Name
		annotations["certmanager.k8s.io/acme-challenge-type"] = "http01"
	}

	return annotations
}

func makeOwnerRef(function *faasv1.FunctionIngress) []metav1.OwnerReference {
	ref := []metav1.OwnerReference{
		*metav1.NewControllerRef(function, schema.GroupVersionKind{
			Group:   faasv1.SchemeGroupVersion.Group,
			Version: faasv1.SchemeGroupVersion.Version,
			Kind:    faasIngressKind,
		}),
	}
	return ref
}
