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

	result := MakeAnnotations(&ingress)

	if _, ok := result["test"]; !ok {
		t.Errorf("Failed to find expected annotation 'test'")
	}
	if _, ok := result["example"]; !ok {
		t.Errorf("Failed to find expected annotation 'example'")
	}
}

func TestMakeAnnotations_IngressClassCanOverride(t *testing.T) {
	wantIngressType := "nginx"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": wantIngressType,
			},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: wantIngressType,
		},
	}

	result := MakeAnnotations(&ingress)

	if val, ok := result["kubernetes.io/ingress.class"]; !ok || val != wantIngressType {
		t.Errorf("Failed to find expected ingress class annotation. Expected '%s' but got '%s'", wantIngressType, val)
	}
}

func TestMakeAnnotations_IngressClassDefaultsToNginx(t *testing.T) {
	wantIngressType := "nginx"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType: wantIngressType,
		},
	}

	result := MakeAnnotations(&ingress)

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

	result := MakeAnnotations(&ingress)
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

	result := MakeAnnotations(&ingress)

	if _, ok := result[defaultRewriteAnnotation]; !ok {
		t.Errorf("Failed to find expected rewrite-target annotation")
	}
}

func TestMakeAnnotations_TraefikAnnotationsAreCorrect(t *testing.T) {
	wantIngressType := "traefik"
	ingress := faasv1.FunctionIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "traefik",
			},
		},
		Spec: faasv1.FunctionIngressSpec{
			IngressType:   wantIngressType,
			Function:      "nodeinfo",
			BypassGateway: false,
			Domain:        "nodeinfo.example.com",
		},
	}

	result := MakeAnnotations(&ingress)
	t.Log(result)

	wantRewriteTarget := "/function/" + ingress.Spec.Function
	if val, ok := result["traefik.ingress.kubernetes.io/rewrite-target"]; !ok || val != wantRewriteTarget {
		t.Errorf("Failed to find expected rewrite target annotation. Expected '%s' but got '%s'", wantRewriteTarget, val)
	}

	wantRuleType := "PathPrefix"
	if val, ok := result["traefik.ingress.kubernetes.io/rule-type"]; !ok || val != wantRuleType {
		t.Errorf("Failed to find expected rule type annotation. Expected '%s' but got '%s'", wantRuleType, val)
	}
}
