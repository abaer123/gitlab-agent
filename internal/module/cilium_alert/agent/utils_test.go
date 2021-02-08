package agent

import (
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/kube_testing"
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestIngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"}},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}, &cnpList)
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
	}

	cnp, err := getPolicy(&flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"unrelatedkey="}},
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"thiskey="}},
	}, &cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_EGRESS,
		Source:           &flow.Endpoint{Labels: []string{"unrelatedkey="}},
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestOtherThanIngressAndEgress(t *testing.T) {
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "Test"},
			},
		},
	}

	cnp, err := getPolicy(&flow.Flow{
		Destination:      &flow.Endpoint{},
		TrafficDirection: flow.TrafficDirection_TRAFFIC_DIRECTION_UNKNOWN,
		Source:           &flow.Endpoint{},
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPFlowEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}

func TestL4OnlyUDPFlowEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyMultiplePortsEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestL4OnlyEgressNoMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Nil(t, cnp)
}

func TestWithL4IngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Test",
			Annotations: map[string]string{"app.gitlab.com/alert": "true"}},
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
	cnpList := v2.CiliumNetworkPolicyList{
		Items: []v2.CiliumNetworkPolicy{*policy},
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
	}, &cnpList)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(cnp, policy, kube_testing.TransformToUnstructured()))
}
