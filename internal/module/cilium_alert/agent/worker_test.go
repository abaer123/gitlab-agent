package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/api/v1/observer"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	cilium_fake "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/fake"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSuccessfulMapping(t *testing.T) {
	for caseNum, matchingData := range matchingData() {
		t.Run(fmt.Sprintf("case %d", caseNum), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cf := cilium_fake.NewSimpleClientset(matchingData.CnpList) // nolint: scopelint
			worker, obsClient, flwClient, mAPI := setupTest(t, cf)
			gomock.InOrder(
				obsClient.EXPECT().
					GetFlows(gomock.Any(), gomock.Any()).
					Return(flwClient, nil),
				flwClient.EXPECT().
					Recv().
					Return(matchingData.FlwResponse, nil), // nolint: scopelint
				mAPI.EXPECT().
					MakeGitLabRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&modagent.GitLabResponse{StatusCode: http.StatusOK}, nil),
				flwClient.EXPECT().
					Recv().
					DoAndReturn(func() (*observer.GetFlowsResponse, error) {
						cancel()
						return nil, errors.New("some error")
					}),
			)
			worker.Run(ctx)
		})
	}
}

func TestNoMatch(t *testing.T) {
	for caseNum, unmatchingData := range unmatchingData() {
		t.Run(fmt.Sprintf("case %d", caseNum), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cf := cilium_fake.NewSimpleClientset(unmatchingData.CnpList) // nolint: scopelint
			worker, obsClient, flwClient, _ := setupTest(t, cf)
			gomock.InOrder(
				obsClient.EXPECT().
					GetFlows(gomock.Any(), gomock.Any()).
					Return(flwClient, nil),
				flwClient.EXPECT().
					Recv().
					Return(unmatchingData.FlwResponse, nil), // nolint: scopelint
				flwClient.EXPECT().
					Recv().
					DoAndReturn(func() (*observer.GetFlowsResponse, error) {
						cancel()
						return nil, errors.New("some error")
					}),
			)
			worker.Run(ctx)
		})
	}
}

func TestJSON(t *testing.T) {
	p1 := payload{
		Alert: alert{
			Flow: (*flowAlias)(&flow.Flow{
				DropReasonDesc: flow.DropReason_POLICY_DENIED,
			}),
			CiliumNetworkPolicy: &v2.CiliumNetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "bla",
					APIVersion: "bla",
				},
			},
		},
	}
	data, err := json.Marshal(p1)
	require.NoError(t, err)

	p2 := payload{}
	err = json.Unmarshal(data, &p2)
	require.NoError(t, err)

	assert.Empty(t, cmp.Diff(p1.Alert.CiliumNetworkPolicy, p2.Alert.CiliumNetworkPolicy))
	assert.Empty(t, cmp.Diff((*flow.Flow)(p1.Alert.Flow), (*flow.Flow)(p2.Alert.Flow), protocmp.Transform()))
}

func setupTest(t *testing.T, cv2 versioned.Interface) (*worker, *MockObserverClient, *MockObserver_GetFlowsClient, *mock_modagent.MockAPI) {
	ctrl := gomock.NewController(t)
	flwClient := NewMockObserver_GetFlowsClient(ctrl)
	obsClient := NewMockObserverClient(ctrl)
	mAPI := mock_modagent.NewMockAPI(ctrl)
	worker := &worker{
		log:            zaptest.NewLogger(t, zaptest.Level(zapcore.DebugLevel)),
		api:            mAPI,
		ciliumClient:   cv2,
		observerClient: obsClient,
		backoff:        testhelpers.NewBackoff(),
		projectId:      21,
	}
	return worker, obsClient, flwClient, mAPI
}

type flwCnpListPair struct {
	FlwResponse *observer.GetFlowsResponse
	CnpList     *v2.CiliumNetworkPolicyList
}

func matchingData() []*flwCnpListPair {
	return []*flwCnpListPair{
		{
			FlwResponse: &observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
				Source: &flow.Endpoint{
					Labels: []string{"otherkey="},
				},
				TrafficDirection: flow.TrafficDirection_INGRESS,
				Destination: &flow.Endpoint{
					Namespace: "ThisNamespace",
					Labels:    []string{"thiskey="},
				},
			}}},
			CnpList: &v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Test",
						Annotations: map[string]string{alertAnnotationKey: "true"},
						Namespace:   "ThisNamespace",
						Labels:      map[string]string{gitLabProjectLabel: "21"},
					},
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
				}},
			},
		},
		{
			FlwResponse: &observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
				Source: &flow.Endpoint{
					Namespace: "ThisNamespace",
					Labels:    []string{"thiskey="},
				},
				TrafficDirection: flow.TrafficDirection_EGRESS,
				Destination: &flow.Endpoint{
					Labels: []string{"otherkey="},
				},
			}}},
			CnpList: &v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Test",
						Annotations: map[string]string{alertAnnotationKey: "true"},
						Namespace:   "ThisNamespace",
						Labels:      map[string]string{gitLabProjectLabel: "21"},
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
				}},
			},
		},
	}
}

func unmatchingData() []*flwCnpListPair {
	return []*flwCnpListPair{
		{
			FlwResponse: &observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
				Source: &flow.Endpoint{
					Labels: []string{"otherkey="},
				},
				TrafficDirection: flow.TrafficDirection_INGRESS,
				Destination: &flow.Endpoint{
					Namespace: "ThisNamespace",
					Labels:    []string{"thiskey="},
				},
			}}},
			CnpList: &v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Test",
						Annotations: map[string]string{alertAnnotationKey: "true"},
						Namespace:   "ThisNamespace",
						Labels:      map[string]string{gitLabProjectLabel: "21"},
					},
					Spec: &api.Rule{
						EndpointSelector: api.NewESFromLabels(labels.NewLabel("notthiskey", "", "any")),
						Ingress: []api.IngressRule{
							{
								IngressCommonRule: api.IngressCommonRule{
									FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
								},
							},
						},
					},
				}},
			},
		},
		{
			FlwResponse: &observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
				Source: &flow.Endpoint{
					Labels: []string{"otherkey="},
				},
				TrafficDirection: flow.TrafficDirection_INGRESS,
				Destination: &flow.Endpoint{
					Namespace: "ThisNamespace",
					Labels:    []string{"thiskey="},
				},
			}}},
			CnpList: &v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Test",
						Annotations: map[string]string{"app.gitlab.com/different": "true"},
						Namespace:   "ThisNamespace",
						Labels:      map[string]string{gitLabProjectLabel: "21"},
					},
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
				}},
			},
		},
		{
			FlwResponse: &observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
				Source: &flow.Endpoint{
					Labels: []string{"otherkey="},
				},
				TrafficDirection: flow.TrafficDirection_INGRESS,
				Destination: &flow.Endpoint{
					Namespace: "ThisNamespace",
					Labels:    []string{"thiskey="},
				},
			}}},
			CnpList: &v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Test",
						Annotations: map[string]string{alertAnnotationKey: "true"},
						Namespace:   "ThisNamespace",
						Labels:      map[string]string{gitLabProjectLabel: "invalid"},
					},
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
				}},
			},
		},
	}
}
