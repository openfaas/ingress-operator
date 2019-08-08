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
