package gitops_agent

import "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"

const (
	allAPIGroups = "*"
	allKinds     = "*"
)

var (
	// Taken from Argo CD, with some additions
	// https://github.com/argoproj/argo-cd/blob/c44074d4d6a6671f6e7be9983fed32f5e1f219e6/util/settings/resources_filter.go#L3-L11
	defaultResourceExclusions = []*agentcfg.ResourceFilterCF{
		{
			ApiGroups: []string{"events.k8s.io", "metrics.k8s.io"},
			Kinds:     []string{allKinds},
		},
		{
			ApiGroups: []string{""},
			Kinds:     []string{"Event", "Node", "Endpoints"},
		},
		{
			ApiGroups: []string{"coordination.k8s.io"},
			Kinds:     []string{"Lease"},
		},
		{
			ApiGroups: []string{"discovery.k8s.io"},
			Kinds:     []string{"EndpointSlice"},
		},
	}
)

type resourcesFilter struct {
	resourceInclusions []*agentcfg.ResourceFilterCF
	resourceExclusions []*agentcfg.ResourceFilterCF
}

func (f resourcesFilter) IsExcludedResource(group, kind, cluster string) bool {
	if resourceMatches(group, kind, f.resourceInclusions) {
		return false
	}
	if resourceMatches(group, kind, defaultResourceExclusions) {
		return true
	}
	if resourceMatches(group, kind, f.resourceExclusions) {
		return true
	}
	return false
}

func resourceMatches(group, kind string, filters []*agentcfg.ResourceFilterCF) bool {
	for _, filter := range filters {
		if groupMatches(group, filter.ApiGroups) && kindMatches(kind, filter.Kinds) {
			return true
		}
	}
	return false
}

func groupMatches(group string, apiGroups []string) bool {
	for _, apiGroupFilter := range apiGroups {
		if apiGroupFilter == allAPIGroups || group == apiGroupFilter {
			return true
		}
	}
	return false
}

func kindMatches(kind string, kinds []string) bool {
	for _, kindFilter := range kinds {
		if kindFilter == allKinds || kind == kindFilter {
			return true
		}
	}
	return false
}
