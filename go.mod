module gitlab.com/gitlab-org/cluster-integration/gitlab-agent

go 1.15

require (
	cloud.google.com/go v0.75.0
	github.com/argoproj/gitops-engine v0.2.1-0.20210108000020-0b4199b00135
	github.com/ash2k/stager v0.2.1
	github.com/bmatcuk/doublestar/v2 v2.0.4
	github.com/cilium/cilium v1.9.1
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1.0.20200107205605-c66185887605
	github.com/envoyproxy/protoc-gen-validate v0.4.2-0.20201217164128-7df253a68e6b
	github.com/go-logr/zapr v0.3.0
	github.com/go-redis/redis/v8 v8.4.8
	github.com/go-redis/redismock/v8 v8.0.3
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.1-0.20200507082539-9abf3eb82b4a
	github.com/opentracing/opentracing-go v1.2.0
	github.com/piotrkowalczuk/promgrpc/v4 v4.0.4
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	gitlab.com/gitlab-org/gitaly v1.87.1-0.20201117220727-89c1ee804f27
	gitlab.com/gitlab-org/labkit v1.3.0
	go.uber.org/zap v1.16.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324
	golang.org/x/tools v0.0.0-20210108195828-e2f9c7f1fc8e
	google.golang.org/api v0.36.0
	google.golang.org/grpc v1.35.0
	google.golang.org/grpc/examples v0.0.0-20200828165940-d8ef479ab79a // indirect
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/cli-runtime v0.20.1
	k8s.io/client-go v0.20.1
	k8s.io/klog/v2 v2.4.0
	nhooyr.io/websocket v1.8.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Cilium depends on v0.19.9 which is incompatible with v0.19.3 which Kubernetes uses. v0.19.8 works fine
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.8
	github.com/golang/mock => github.com/golang/mock v1.4.4-0.20201210203420-1fe605df5e5f
	github.com/optiopay/kafka => github.com/cilium/kafka v0.0.0-20180809090225-01ce283b732b
	// same version as used by rules_go to maintain compatibility with patches - see the WORKSPACE file
	golang.org/x/tools => golang.org/x/tools v0.0.0-20201201192219-a1b87a1c0de4

	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.20.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.1
	k8s.io/apiserver => k8s.io/apiserver v0.20.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.1
	k8s.io/client-go => k8s.io/client-go v0.20.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.1
	k8s.io/code-generator => k8s.io/code-generator v0.20.1
	k8s.io/component-base => k8s.io/component-base v0.20.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.20.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.20.1
	k8s.io/cri-api => k8s.io/cri-api v0.20.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.1
	k8s.io/kubectl => k8s.io/kubectl v0.20.1
	k8s.io/kubelet => k8s.io/kubelet v0.20.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.1
	k8s.io/metrics => k8s.io/metrics v0.20.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.20.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.1
)
