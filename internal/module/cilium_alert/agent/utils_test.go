package agent

import (
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
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
	assert.Equal(t, cnp, policy)
}

func TestIngressNotMatched(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Ingress: []api.IngressRule{
				{
					FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
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
	assert.NotEqual(t, cnp, policy)
}

func TestEgressMatch(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
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
	assert.Equal(t, cnp, policy)
}

func TestEgressNotMatched(t *testing.T) {
	policy := &v2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "Test"},
		Spec: &api.Rule{
			EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
			Egress: []api.EgressRule{
				{
					ToEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
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
	assert.NotEqual(t, cnp, policy)
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
	assert.Nil(t, err)
}
