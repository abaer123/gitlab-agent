package agent

import (
	"fmt"
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/cilium/cilium/pkg/policy/api/kafka"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/kube_testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressMatchButWithoutAnnotation(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestIngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"}},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestIngressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"unrelatedkey="}},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					EgressCommonRule: api.EgressCommonRule{
						ToEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					EgressCommonRule: api.EgressCommonRule{
						ToEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"unrelatedkey="}},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestOtherThanIngressAndEgress(t *testing.T) {
	cnpList := []interface{}{
		&v2.CiliumNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		},
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination:      &flow.Endpoint{},
		TrafficDirection: flow.TrafficDirection_TRAFFIC_DIRECTION_UNKNOWN,
		Source:           &flow.Endpoint{},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "5000",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 80,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "5000",
									Protocol: api.ProtoUDP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 5000,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPFlowEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "5000",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_UDP{
				UDP: &flow.UDP{
					SourcePort:      8081,
					DestinationPort: 5000,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPFlowEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "5000",
									Protocol: api.ProtoUDP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_UDP{
				UDP: &flow.UDP{
					SourcePort:      8081,
					DestinationPort: 5000,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyMultiplePortsEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "5000",
									Protocol: api.ProtoTCP,
								},
								{
									Port:     "80",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 80,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"},
		},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "80",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 80,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestWithL4IngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"}},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "80",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 80,
				},
			},
		},
	}, cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestWithL7IngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"}},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "80",
									Protocol: api.ProtoTCP,
								},
							},
							Rules: &api.L7Rules{
								HTTP: []api.PortRuleHTTP{
									{
										Headers: []string{
											"first: label",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	flow := flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"thiskey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"thiskey="},
		},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      8081,
					DestinationPort: 80,
				},
			},
		},
		L7: &flow.Layer7{
			Record: &flow.Layer7_Http{
				Http: &flow.HTTP{
					Headers: []*flow.HTTPHeader{
						{
							Key:   "first",
							Value: "unexpected",
						},
					},
				},
			},
		},
	}

	assert.NotNil(t, flow.GetL7())
	rules, err := policy.Parse()
	require.NoError(t, err)
	assert.True(t, l7Applicable(rules, flow.GetTrafficDirection()))

	cnp, err := getPolicy(&flow, cnpList)
	require.NoError(t, err)
	assert.NotNil(t, cnp)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestWithL7EgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{alertAnnotationKey: "true"}},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					EgressCommonRule: api.EgressCommonRule{
						ToEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("thiskey", "", "any"))},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "80",
									Protocol: api.ProtoTCP,
								},
							},
							Rules: &api.L7Rules{
								HTTP: []api.PortRuleHTTP{
									{
										Headers: []string{
											"first: label",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	cnpList := []interface{}{
		policy,
	}

	flow := flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"thiskey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"thiskey="},
		},
		L4: &flow.Layer4{
			Protocol: &flow.Layer4_TCP{
				TCP: &flow.TCP{
					SourcePort:      80,
					DestinationPort: 80,
				},
			},
		},
		L7: &flow.Layer7{
			Record: &flow.Layer7_Http{
				Http: &flow.HTTP{
					Headers: []*flow.HTTPHeader{
						{
							Key:   "first",
							Value: "label",
						},
					},
				},
			},
		},
	}

	assert.NotNil(t, flow.GetL7())
	rules, err := policy.Parse()
	require.NoError(t, err)
	assert.True(t, l7Applicable(rules, flow.GetTrafficDirection()))

	cnp, err := getPolicy(&flow, cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestIncludeHeaderMatch(t *testing.T) {
	ntwHrd := []string{"first: label", "second: label"}
	flwHrds := []*flow.HTTPHeader{
		{
			Key:   "first",
			Value: "label",
		},
		{
			Key:   "second",
			Value: "label",
		},
		{
			Key:   "third",
			Value: "label",
		},
	}
	assert.True(t, includeHeader(ntwHrd, flwHrds))
}

func TestIncludeHeaderNoMatch(t *testing.T) {
	ntwHrd := []string{"first: label", "second: label", "third: label"}
	flwHrds := []*flow.HTTPHeader{
		{
			Key:   "first",
			Value: "label",
		},
		{
			Key:   "second",
			Value: "label",
		},
	}
	assert.False(t, includeHeader(ntwHrd, flwHrds))
}

func TestEmptyIncludeHeaderNoMatch(t *testing.T) {
	ntwHrd := []string{"first: label", "second: label", "third: label"}
	flwHrds := []*flow.HTTPHeader{}
	assert.False(t, includeHeader(ntwHrd, flwHrds))
}

func TestMalformedIncludeHeaderNoMatch(t *testing.T) {
	ntwHrd := []string{"first label", "second: label"}
	flwHrds := []*flow.HTTPHeader{
		{
			Key:   "first",
			Value: "label",
		},
		{
			Key:   "second",
			Value: "label",
		},
	}
	assert.False(t, includeHeader(ntwHrd, flwHrds))
}

func TestL7Applicable(t *testing.T) {
	ingRules := api.Rules{
		{
			Ingress: []api.IngressRule{
				{
					ToPorts: api.PortRules{
						{
							Rules: &api.L7Rules{
								HTTP: []api.PortRuleHTTP{
									{},
								},
							},
						},
					},
				},
			},
		},
	}
	egrRules := api.Rules{
		{
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Rules: &api.L7Rules{
								HTTP: []api.PortRuleHTTP{
									{},
								},
							},
						},
					},
				},
			},
		},
	}
	assert.True(t, l7Applicable(ingRules, flow.TrafficDirection_INGRESS))
	assert.True(t, l7Applicable(egrRules, flow.TrafficDirection_EGRESS))
}

func TestNotL7Applicable(t *testing.T) {
	ingRules := api.Rules{
		{
			Ingress: []api.IngressRule{
				{
					ToPorts: api.PortRules{
						{
							Rules: &api.L7Rules{
								Kafka: []kafka.PortRule{},
							},
						},
					},
				},
			},
		},
	}
	egrRules := api.Rules{
		{
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{
							Rules: &api.L7Rules{},
						},
					},
				},
			},
		},
	}

	noRules := api.Rules{
		{
			Egress: []api.EgressRule{
				{
					ToPorts: api.PortRules{
						{},
					},
				},
			},
		},
	}

	assert.False(t, l7Applicable(ingRules, flow.TrafficDirection_INGRESS))
	assert.False(t, l7Applicable(egrRules, flow.TrafficDirection_EGRESS))
	assert.False(t, l7Applicable(noRules, flow.TrafficDirection_EGRESS))
}

func TestL7DirectMatch(t *testing.T) {
	toPorts := api.PortRules{
		{
			Rules: &api.L7Rules{
				HTTP: []api.PortRuleHTTP{
					{
						Headers: []string{"first: label"},
						Method:  "GET",
						Host:    "foo.com",
						Path:    "/path",
					},
				},
			},
		},
	}
	rules := api.Rules{
		{
			Ingress: []api.IngressRule{
				{
					ToPorts: toPorts,
				},
			},
		},
		{
			Egress: []api.EgressRule{
				{
					ToPorts: toPorts,
				},
			},
		},
	}
	layer7 := &flow.Layer7{
		Record: &flow.Layer7_Http{
			Http: &flow.HTTP{
				Headers: []*flow.HTTPHeader{
					{
						Key:   "first",
						Value: "label",
					},
					{
						Key:   "second",
						Value: "label",
					},
				},
				Method: "GET",
				Url:    "https://foo.com/path",
			},
		},
	}
	flws := []*flow.Flow{
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
			L7:               layer7,
		},
		{
			TrafficDirection: flow.TrafficDirection_EGRESS,
			L7:               layer7,
		},
	}
	for _, flw := range flws {
		t.Run(fmt.Sprintf("Test %v", flw.GetTrafficDirection()), func(t *testing.T) {
			assert.True(t, matchL7Policy(rules, flw)) // nolint: scopelint
		})
	}
}

func TestL7DirectUnMatch(t *testing.T) {
	toPorts := api.PortRules{
		{
			Rules: &api.L7Rules{
				HTTP: []api.PortRuleHTTP{
					{
						Headers: []string{"third: label"},
						Method:  "GET",
						Host:    "foo.com$",
					},
				},
			},
		},
	}
	rules := api.Rules{
		{
			Ingress: []api.IngressRule{
				{
					ToPorts: toPorts,
				},
			},
		},
		{
			Egress: []api.EgressRule{
				{
					ToPorts: toPorts,
				},
			},
		},
	}

	for idx, flw := range unmatchFlws() {
		t.Run(fmt.Sprintf("Test %v", idx), func(t *testing.T) {
			assert.False(t, matchL7Policy(rules, flw)) // nolint: scopelint
		})
	}
}

func unmatchFlws() []*flow.Flow {
	return []*flow.Flow{
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{
					Http: &flow.HTTP{
						Headers: []*flow.HTTPHeader{
							{
								Key:   "third",
								Value: "label",
							},
						},
						Method: "GET",
						Url:    "https://foo.com.br",
					},
				},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{
					Http: &flow.HTTP{
						Headers: []*flow.HTTPHeader{
							{
								Key:   "third",
								Value: "label",
							},
						},
						Method: "POST",
						Url:    "https://foo.com",
					},
				},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_EGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{
					Http: &flow.HTTP{
						Headers: []*flow.HTTPHeader{
							{
								Key:   "first",
								Value: "label",
							},
							{
								Key:   "second",
								Value: "label",
							},
						},
						Method: "GET",
						Url:    "https://foo.com",
					},
				},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{
					Http: &flow.HTTP{
						Headers: []*flow.HTTPHeader{},
						Method:  "GET",
						Url:     "https://foo.com",
					},
				},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_EGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{
					Http: &flow.HTTP{},
				},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
			L7: &flow.Layer7{
				Record: &flow.Layer7_Http{},
			},
		},
		{
			TrafficDirection: flow.TrafficDirection_EGRESS,
			L7:               &flow.Layer7{},
		},
		{
			TrafficDirection: flow.TrafficDirection_INGRESS,
		},
	}
}
