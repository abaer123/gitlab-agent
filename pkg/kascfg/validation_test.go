package kascfg

import (
	"testing"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestValidation_Valid(t *testing.T) {
	tests := []testhelpers.ValidTestcase{
		{
			Name: "minimal",
			Valid: &ConfigurationFile{
				Gitlab: &GitLabCF{
					Address:                  "http://localhost:8080",
					AuthenticationSecretFile: "/some/file",
				},
			},
		},
		{
			Name: "GitopsCF",
			Valid: &GitopsCF{
				ProjectInfoCacheTtl:      durationpb.New(0), // zero means "disabled"
				MaxManifestFileSize:      0,                 // zero means "use default value"
				MaxTotalManifestFileSize: 0,                 // zero means "use default value"
				MaxNumberOfPaths:         0,                 // zero means "use default value"
				MaxNumberOfFiles:         0,                 // zero means "use default value"
			},
		},
		{
			Name: "AgentCF",
			Valid: &AgentCF{
				InfoCacheTtl: durationpb.New(0), // zero means "disabled"
			},
		},
		{
			Name: "ObservabilityCF",
			Valid: &ObservabilityCF{
				UsageReportingPeriod: durationpb.New(0), // zero means "disabled"
			},
		},
		{
			Name: "TokenBucketRateLimitCF",
			Valid: &TokenBucketRateLimitCF{
				RefillRatePerSecond: 0, // zero means "use default value"
				BucketSize:          0, // zero means "use default value"
			},
		},
		{
			Name: "RedisCF",
			Valid: &RedisCF{
				RedisConfig: &RedisCF_Server{
					Server: &RedisServerCF{
						Address: "//path/to/socket.sock",
					},
				},
				PoolSize:  0,  // zero means "use default value"
				KeyPrefix: "", // empty means "use default value"
			},
		},
		{
			Name: "RedisCF",
			Valid: &RedisCF{
				RedisConfig: &RedisCF_Server{
					Server: &RedisServerCF{
						Address: "address:6380",
					},
				},
				PoolSize:  0,  // zero means "use default value"
				KeyPrefix: "", // empty means "use default value"
			},
		},
		{
			Name: "RedisCF",
			Valid: &RedisCF{
				RedisConfig: &RedisCF_Server{
					Server: &RedisServerCF{
						Address: "127.0.0.1:6380",
					},
				},
				PoolSize:  0,  // zero means "use default value"
				KeyPrefix: "", // empty means "use default value"
			},
		},
		{
			Name: "AgentConfigurationCF",
			Valid: &AgentConfigurationCF{
				MaxConfigurationFileSize: 0, // zero means "use default value"
			},
		},
		{
			Name: "ListenAgentCF",
			Valid: &ListenAgentCF{
				ConnectionsPerTokenPerMinute: 0, // zero means "use default value"
			},
		},
	}
	testhelpers.AssertValid(t, tests)
}

func TestValidation_Invalid(t *testing.T) {
	tests := []testhelpers.InvalidTestcase{
		{
			Name:      "zero GitopsCF.PollPeriod",
			ErrString: "invalid GitopsCF.PollPeriod: value must be greater than 0s",
			Invalid: &GitopsCF{
				PollPeriod: durationpb.New(0),
			},
		},
		{
			Name:      "negative GitopsCF.PollPeriod",
			ErrString: "invalid GitopsCF.PollPeriod: value must be greater than 0s",
			Invalid: &GitopsCF{
				PollPeriod: durationpb.New(-1),
			},
		},
		{
			Name:      "negative GitopsCF.ProjectInfoCacheTtl",
			ErrString: "invalid GitopsCF.ProjectInfoCacheTtl: value must be greater than or equal to 0s",
			Invalid: &GitopsCF{
				ProjectInfoCacheTtl: durationpb.New(-1),
			},
		},
		{
			Name:      "zero GitopsCF.ProjectInfoCacheErrorTtl",
			ErrString: "invalid GitopsCF.ProjectInfoCacheErrorTtl: value must be greater than 0s",
			Invalid: &GitopsCF{
				ProjectInfoCacheErrorTtl: durationpb.New(0),
			},
		},
		{
			Name:      "negative GitopsCF.ProjectInfoCacheErrorTtl",
			ErrString: "invalid GitopsCF.ProjectInfoCacheErrorTtl: value must be greater than 0s",
			Invalid: &GitopsCF{
				ProjectInfoCacheErrorTtl: durationpb.New(-1),
			},
		},
		{
			Name:      "negative AgentCF.InfoCacheTtl",
			ErrString: "invalid AgentCF.InfoCacheTtl: value must be greater than or equal to 0s",
			Invalid: &AgentCF{
				InfoCacheTtl: durationpb.New(-1),
			},
		},
		{
			Name:      "zero AgentCF.InfoCacheErrorTtl",
			ErrString: "invalid AgentCF.InfoCacheErrorTtl: value must be greater than 0s",
			Invalid: &AgentCF{
				InfoCacheErrorTtl: durationpb.New(0),
			},
		},
		{
			Name:      "negative AgentCF.InfoCacheErrorTtl",
			ErrString: "invalid AgentCF.InfoCacheErrorTtl: value must be greater than 0s",
			Invalid: &AgentCF{
				InfoCacheErrorTtl: durationpb.New(-1),
			},
		},
		{
			Name:      "zero AgentConfigurationCF.PollPeriod",
			ErrString: "invalid AgentConfigurationCF.PollPeriod: value must be greater than 0s",
			Invalid: &AgentConfigurationCF{
				PollPeriod: durationpb.New(0),
			},
		},
		{
			Name:      "negative AgentConfigurationCF.PollPeriod",
			ErrString: "invalid AgentConfigurationCF.PollPeriod: value must be greater than 0s",
			Invalid: &AgentConfigurationCF{
				PollPeriod: durationpb.New(-1),
			},
		},
		{
			Name:      "negative ObservabilityCF.UsageReportingPeriod",
			ErrString: "invalid ObservabilityCF.UsageReportingPeriod: value must be greater than or equal to 0s",
			Invalid: &ObservabilityCF{
				UsageReportingPeriod: durationpb.New(-1),
			},
		},
		{
			Name:      "negative TokenBucketRateLimitCF.RefillRatePerSecond",
			ErrString: "invalid TokenBucketRateLimitCF.RefillRatePerSecond: value must be greater than or equal to 0",
			Invalid: &TokenBucketRateLimitCF{
				RefillRatePerSecond: -1,
			},
		},
		{
			Name:      "zero RedisCF.DialTimeout",
			ErrString: "invalid RedisCF.DialTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				DialTimeout: durationpb.New(0),
			},
		},
		{
			Name:      "negative RedisCF.DialTimeout",
			ErrString: "invalid RedisCF.DialTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				DialTimeout: durationpb.New(-1),
			},
		},
		{
			Name:      "zero RedisCF.ReadTimeout",
			ErrString: "invalid RedisCF.ReadTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				ReadTimeout: durationpb.New(0),
			},
		},
		{
			Name:      "negative RedisCF.ReadTimeout",
			ErrString: "invalid RedisCF.ReadTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				ReadTimeout: durationpb.New(-1),
			},
		},
		{
			Name:      "zero RedisCF.WriteTimeout",
			ErrString: "invalid RedisCF.WriteTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				WriteTimeout: durationpb.New(0),
			},
		},
		{
			Name:      "negative RedisCF.WriteTimeout",
			ErrString: "invalid RedisCF.WriteTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				WriteTimeout: durationpb.New(-1),
			},
		},
		{
			Name:      "zero RedisCF.IdleTimeout",
			ErrString: "invalid RedisCF.IdleTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				IdleTimeout: durationpb.New(0),
			},
		},
		{
			Name:      "negative RedisCF.IdleTimeout",
			ErrString: "invalid RedisCF.IdleTimeout: value must be greater than 0s",
			Invalid: &RedisCF{
				IdleTimeout: durationpb.New(-1),
			},
		},
		{
			Name:      "missing RedisCF.RedisConfig",
			ErrString: "invalid RedisCF.RedisConfig: value is required",
			Invalid:   &RedisCF{},
		},
		{
			Name:      "empty RedisServerCF.Address",
			ErrString: "invalid RedisServerCF.Address: value length must be at least 1 runes",
			Invalid:   &RedisServerCF{},
		},
		{
			Name:      "empty RedisSentinelCF.MasterName",
			ErrString: "invalid RedisSentinelCF.MasterName: value length must be at least 1 runes",
			Invalid: &RedisSentinelCF{
				Addresses: []string{"1:2"},
			},
		},
		{
			Name:      "empty RedisSentinelCF.Addresses",
			ErrString: "invalid RedisSentinelCF.Addresses: value must contain at least 1 item(s)",
			Invalid: &RedisSentinelCF{
				MasterName: "bla",
			},
		},
		{
			Name:      "zero ListenAgentCF.MaxConnectionAge",
			ErrString: "invalid ListenAgentCF.MaxConnectionAge: value must be greater than 0s",
			Invalid: &ListenAgentCF{
				MaxConnectionAge: durationpb.New(0),
			},
		},
		{
			Name:      "negative ListenAgentCF.MaxConnectionAge",
			ErrString: "invalid ListenAgentCF.MaxConnectionAge: value must be greater than 0s",
			Invalid: &ListenAgentCF{
				MaxConnectionAge: durationpb.New(-1),
			},
		},
		{
			Name:      "empty GitLabCF.Address",
			ErrString: "invalid GitLabCF.Address: value length must be at least 1 runes",
			Invalid: &GitLabCF{
				AuthenticationSecretFile: "/some/file",
			},
		},
		{
			Name:      "relative GitLabCF.Address",
			ErrString: "invalid GitLabCF.Address: value must be absolute",
			Invalid: &GitLabCF{
				Address:                  "/path",
				AuthenticationSecretFile: "/some/file",
			},
		},
		{
			Name:      "empty GitLabCF.AuthenticationSecretFile",
			ErrString: "invalid GitLabCF.AuthenticationSecretFile: value length must be at least 1 runes",
			Invalid: &GitLabCF{
				Address: "http://localhost:8080",
			},
		},
		// TODO uncomment when Redis becomes a hard dependency
		//{
		//	Name:      "missing ConfigurationFile.Redis",
		//	ErrString: "invalid ConfigurationFile.Redis: value is required",
		//	Invalid: &ConfigurationFile{
		//		Gitlab: &GitLabCF{
		//			Address:                  "http://localhost:8080",
		//			AuthenticationSecretFile: "/some/file",
		//		},
		//	},
		//},
		{
			Name:      "missing ConfigurationFile.Gitlab",
			ErrString: "invalid ConfigurationFile.Gitlab: value is required",
			Invalid:   &ConfigurationFile{},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
