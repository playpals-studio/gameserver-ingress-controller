package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"testing"
)

func Test_NewIngress_DomainRoutingMode(t *testing.T) {
	testCases := []struct {
		name          string
		terminateTLS  bool
		certTLSIssuer string
	}{
		{
			name:          "terminate tls",
			terminateTLS:  true,
			certTLSIssuer: "selfSigned",
		},
		{
			name:         "do not terminate tls",
			terminateTLS: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domain := "foo.bar"
			gs := newGameServer(map[string]string{
				gameserver.OctopsAnnotationIngressMode:   string(gameserver.IngressRoutingModeDomain),
				gameserver.OctopsAnnotationIngressDomain: domain,
				gameserver.OctopsAnnotationTerminateTLS:  strconv.FormatBool(tc.terminateTLS),
				gameserver.OctopsAnnotationIssuerName:    tc.certTLSIssuer,
			})

			mode := gameserver.GetIngressRoutingMode(gs)
			issuerName := gameserver.GetTLSCertIssuer(gs)
			host := fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain])
			ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
			tls := newIngressTLS(host, gs.Name)
			rules := newIngressRule(host, "/", gs.Name, gameserver.GetGameServerPort(gs).Port)

			opts := []IngressOption{
				WithIngressRule(mode),
				WithTLS(mode),
				WithTLSCertIssuer(issuerName),
			}
			ig, err := newIngress(gs, opts...)

			require.NoError(t, err)
			require.Equal(t, gs.Name, ig.Name)
			require.Equal(t, gameserver.GetGameServerPort(gs).Port, ig.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Number)
			require.Contains(t, ig.Labels, gameserver.AgonesGameServerNameLabel)
			require.Equal(t, ig.Labels[gameserver.AgonesGameServerNameLabel], gs.Name)
			require.Equal(t, []metav1.OwnerReference{*ref}, ig.OwnerReferences)
			require.Equal(t, tls, ig.Spec.TLS)
			require.Equal(t, rules, ig.Spec.Rules)

			if tc.terminateTLS {
				require.Contains(t, ig.Annotations, gameserver.CertManagerAnnotationIssuer)
				require.Equal(t, issuerName, ig.Annotations[gameserver.CertManagerAnnotationIssuer])
			} else {
				require.NotContains(t, ig.Annotations, gameserver.CertManagerAnnotationIssuer)
				require.Empty(t, ig.Annotations[gameserver.CertManagerAnnotationIssuer])
			}
		})
	}

}

func Test_NewIngress_PathRoutingMode(t *testing.T) {
	testCases := []struct {
		name          string
		terminateTLS  bool
		certTLSIssuer string
	}{
		{
			name:          "terminate tls",
			terminateTLS:  true,
			certTLSIssuer: "selfSigned",
		},
		{
			name:         "do not terminate tls",
			terminateTLS: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fqdn := "servers.foo.bar"
			gs := newGameServer(map[string]string{
				gameserver.OctopsAnnotationIngressFQDN:  fqdn,
				gameserver.OctopsAnnotationIngressMode:  string(gameserver.IngressRoutingModePath),
				gameserver.OctopsAnnotationTerminateTLS: strconv.FormatBool(tc.terminateTLS),
				gameserver.OctopsAnnotationIssuerName:   tc.certTLSIssuer,
			})

			mode := gameserver.GetIngressRoutingMode(gs)
			issuerName := gameserver.GetTLSCertIssuer(gs)

			ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
			tls := newIngressTLS(fqdn, gs.Name)
			rules := newIngressRule(gs.Annotations[gameserver.OctopsAnnotationIngressFQDN], "/"+gs.Name, gs.Name, gameserver.GetGameServerPort(gs).Port)

			opts := []IngressOption{
				WithIngressRule(mode),
				WithTLS(mode),
				WithTLSCertIssuer(issuerName),
			}
			ig, err := newIngress(gs, opts...)

			require.NoError(t, err)
			require.Equal(t, gs.Name, ig.Name)
			require.Equal(t, gameserver.GetGameServerPort(gs).Port, ig.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Number)
			require.Equal(t, "/"+gs.Name, ig.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path)
			require.Contains(t, ig.Labels, gameserver.AgonesGameServerNameLabel)
			require.Equal(t, ig.Labels[gameserver.AgonesGameServerNameLabel], gs.Name)
			require.Equal(t, []metav1.OwnerReference{*ref}, ig.OwnerReferences)
			require.Equal(t, tls, ig.Spec.TLS)
			require.Equal(t, rules, ig.Spec.Rules)

			if tc.terminateTLS {
				require.Contains(t, ig.Annotations, gameserver.CertManagerAnnotationIssuer)
				require.Equal(t, issuerName, ig.Annotations[gameserver.CertManagerAnnotationIssuer])
			} else {
				require.NotContains(t, ig.Annotations, gameserver.CertManagerAnnotationIssuer)
				require.Empty(t, ig.Annotations[gameserver.CertManagerAnnotationIssuer])
			}
		})
	}
}

func newGameServer(annotations map[string]string) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "simple-gameserver",
			Namespace:   "default",
			Annotations: annotations,
		},
		Status: agonesv1.GameServerStatus{
			Ports: []agonesv1.GameServerStatusPort{
				{
					Port: 7771,
				},
			},
		},
	}
}