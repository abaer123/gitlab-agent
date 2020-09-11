package agentk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
)

func TestDefaultExclusions(t *testing.T) {
	for _, cfg := range defaultResourceExclusions {
		for _, group := range cfg.ApiGroups {
			for _, kind := range cfg.Kinds {
				t.Run(fmt.Sprintf("%s/%s", group, kind), func(t *testing.T) {
					assert.True(t, resourcesFilter{}.IsExcludedResource(group, kind, "")) // nolint: scopelint
				})
			}
		}
	}
}

func TestDefaultBehaviorIsInclude(t *testing.T) {
	assert.False(t, resourcesFilter{}.IsExcludedResource("", "ConfigMap", ""))
}

func TestExcludeWildcardGroup(t *testing.T) {
	assert.True(t, resourcesFilter{
		resourceExclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{allAPIGroups},
				Kinds:     []string{"ConfigMap"},
			},
		},
	}.IsExcludedResource("group1", "ConfigMap", ""))
}

func TestIncludeWildcardGroup(t *testing.T) {
	assert.False(t, resourcesFilter{
		resourceExclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{allAPIGroups},
				Kinds:     []string{"ConfigMap"},
			},
		},
	}.IsExcludedResource("group1", "Secret", ""))
}

func TestExcludeWildcardKind(t *testing.T) {
	assert.True(t, resourcesFilter{
		resourceExclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{"group1"},
				Kinds:     []string{allKinds},
			},
		},
	}.IsExcludedResource("group1", "ConfigMap", ""))
}

func TestIncludeWildcardKind(t *testing.T) {
	assert.False(t, resourcesFilter{
		resourceExclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{"group1"},
				Kinds:     []string{allKinds},
			},
		},
	}.IsExcludedResource("group2", "ConfigMap", ""))
}

func TestIncludeException(t *testing.T) {
	filter := resourcesFilter{
		resourceInclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{"group1"},
				Kinds:     []string{"ConfigMap"},
			},
		},
		resourceExclusions: []*agentcfg.ResourceFilterCF{
			{
				ApiGroups: []string{allAPIGroups},
				Kinds:     []string{allKinds},
			},
		},
	}
	assert.False(t, filter.IsExcludedResource("group1", "ConfigMap", ""))
	assert.True(t, filter.IsExcludedResource("group1", "Secret", ""))
}
