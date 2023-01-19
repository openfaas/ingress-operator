package v1beta1

import (
	"reflect"
	"testing"

	v1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	faasv1 "github.com/openfaas/ingress-operator/pkg/apis/openfaas/v1"
	"github.com/openfaas/ingress-operator/pkg/controller"
)

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

	if gotPort.IntValue() != controller.OpenfaasWorkloadPort {
		t.Errorf("want port %d, but got %d", controller.OpenfaasWorkloadPort, gotPort.IntValue())
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

func Test_makeRules_Traefik_NestedPath_TrimsRegex_And_TrailingSlash(t *testing.T) {
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

	wantPath := "/v1/profiles/view"
	gotPath := rules[0].HTTP.Paths[0].Path
	if gotPath != wantPath {
		t.Errorf("want path %s, but got %s", wantPath, gotPath)
	}
}

func Test_makTLS(t *testing.T) {

	cases := []struct {
		name     string
		fni      *faasv1.FunctionIngress
		expected []v1beta1.IngressTLS
	}{
		{
			name:     "tls disabled results in empty tls config",
			fni:      &faasv1.FunctionIngress{Spec: faasv1.FunctionIngressSpec{TLS: &faasv1.FunctionIngressTLS{Enabled: false}}},
			expected: []v1beta1.IngressTLS{},
		},
		{
			name: "tls enabled creates TLS object with correct host and secret with matching the host",
			fni: &faasv1.FunctionIngress{
				Spec: faasv1.FunctionIngressSpec{
					Domain: "foo.example.com",
					TLS: &faasv1.FunctionIngressTLS{
						Enabled: true,
						IssuerRef: faasv1.ObjectReference{
							Name: "test-issuer",
							Kind: "ClusterIssuer",
						},
					},
				},
			},
			expected: []v1beta1.IngressTLS{
				{
					SecretName: "foo.example.com-cert",
					Hosts: []string{
						"foo.example.com",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := makeTLS(tc.fni)
			if !reflect.DeepEqual(tc.expected, got) {
				t.Fatalf("want tls config %v, got %v", tc.expected, got)
			}
		})
	}
}
