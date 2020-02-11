package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	faasv1 "github.com/openfaas-incubator/ingress-operator/pkg/apis/openfaas/v1alpha2"
)

func TestMakeAnnotations_AnnotationsCopied(t *testing.T) {
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"test":    "test",
				"example": "example",
			},
		},
	}

	result := makeAnnotations(&ingress)

	if _, ok := result["test"]; !ok {
		t.Errorf("Failed to find expected annotation 'test'")
	}
	if _, ok := result["example"]; !ok {
		t.Errorf("Failed to find expected annotation 'example'")
	}
}

func TestMakeAnnotations_IngressClass(t *testing.T) {
	wantIngressType := "nginx"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "traefik",
			},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: wantIngressType,
		},
	}

	result := makeAnnotations(&ingress)

	if val, ok := result["kubernetes.io/ingress.class"]; !ok || val != wantIngressType {
		t.Errorf("Failed to find expected ingress class annotation. Expected '%s' but got '%s'", wantIngressType, val)
	}
}

func TestMakeAnnotations_ByPassRemovesRewriteTarget(t *testing.T) {
	wantIngressType := "nginx"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType:   wantIngressType,
			Function:      "nodeinfo",
			BypassGateway: true,
			Domain:        "nodeinfo.example.com",
		},
	}

	result := makeAnnotations(&ingress)
	t.Log(result)
	if val, ok := result["nginx.ingress.kubernetes.io/rewrite-target"]; ok {
		t.Errorf("No rewrite annotations should be given, but got: %s", val)
	}
}

func TestMakeAnnotations_IngressClassAdditionalAnnotations(t *testing.T) {
	defaultRewriteAnnotation := "nginx.ingress.kubernetes.io/rewrite-target"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: "nginx",
		},
	}

	result := makeAnnotations(&ingress)

	if _, ok := result[defaultRewriteAnnotation]; !ok {
		t.Errorf("Failed to find expected rewrite-target annotation")
	}
}

func Test_makeRules_Nginx_RootPath_HasRegex(t *testing.T) {
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: "nginx",
		},
	}

	rules := makeRules(&ingress)

	if len(rules) == 0 {
		t.Errorf("Ingress should give at least one rule")
		t.Fail()
	}

	wantPath := "/(.*)"
	gotPath := rules[0].HTTP.Paths[0].Path

	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}

	gotPort := rules[0].HTTP.Paths[0].Backend.ServicePort

	if gotPort.IntValue() != openfaasWorkloadPort {
		t.Errorf("want port %d, but got %d", openfaasWorkloadPort, gotPort.IntValue())
	}
}

func Test_makeRules_Nginx_RootPath_IsRootWithBypassMode(t *testing.T) {
	wantFunction := "nodeinfo"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			BypassGateway: true,
			IngressType:   "nginx",
			Function:      "nodeinfo",
			// Path:          "/",
		},
	}

	rules := makeRules(&ingress)

	if len(rules) == 0 {
		t.Errorf("Ingress should give at least one rule")
		t.Fail()
	}

	wantPath := "/"
	gotPath := rules[0].HTTP.Paths[0].Path

	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}

	gotHost := rules[0].HTTP.Paths[0].Backend.ServiceName

	if gotHost != wantFunction {
		t.Errorf("want host to be function: %s, but got %s", wantFunction, gotHost)
	}
}

func Test_makeRules_Nginx_PathOverride(t *testing.T) {
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: "nginx",
			Path:        "/v1/profiles/view/(.*)",
		},
	}

	rules := makeRules(&ingress)

	if len(rules) == 0 {
		t.Errorf("Ingress should give at least one rule")
		t.Fail()
	}

	wantPath := ingress.Spec.Path
	gotPath := rules[0].HTTP.Paths[0].Path

	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}
}

func Test_makeRules_Traefik_RootPath_TrimsRegex(t *testing.T) {
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: "traefik",
		},
	}

	rules := makeRules(&ingress)

	if len(rules) == 0 {
		t.Errorf("Ingress should give at least one rule")
		t.Fail()
	}

	wantPath := "/"
	gotPath := rules[0].HTTP.Paths[0].Path
	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}
}

func Test_makeRules_Traefik_NestedPath_TrimsRegex(t *testing.T) {
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: "traefik",
			Path:        "/v1/profiles/view/(.*)",
		},
	}

	rules := makeRules(&ingress)

	if len(rules) == 0 {
		t.Errorf("Ingress should give at least one rule")
		t.Fail()
	}

	wantPath := "/v1/profiles/view/"
	gotPath := rules[0].HTTP.Paths[0].Path
	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}
}
