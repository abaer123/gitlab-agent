module gitlab.com/gitlab-org/cluster-integration/gitlab-agent

go 1.15

require (
	github.com/argoproj/gitops-engine v0.1.3-0.20201013235340-a1dc4c598b93
	github.com/ash2k/stager v0.2.0
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1.0.20200107205605-c66185887605
	github.com/getsentry/sentry-go v0.7.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.1-0.20200507082539-9abf3eb82b4a
	// Pick up https://github.com/moby/term/commit/3e73b07ecbf5dc7b59fcecc783c4988c6b5aa767 which fixes the breakage
	// caused by https://github.com/golang/sys/commit/6fcdbc0bbc04
	// This can be removed once Kubernetes bumps the dependency.
	github.com/moby/term v0.0.0-20200915141129-7f0af18e79f2 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/piotrkowalczuk/promgrpc/v4 v4.0.2
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gitlab.com/gitlab-org/gitaly v1.87.1-0.20201016033652-3bdd23173595
	gitlab.com/gitlab-org/labkit v0.0.0-20201014124351-eb1fe6499318
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	golang.org/x/tools v0.0.0-20200823205832-c024452afbcd
	google.golang.org/grpc v1.33.0
	google.golang.org/grpc/examples v0.0.0-20200828165940-d8ef479ab79a // indirect
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/cli-runtime v0.19.2
	k8s.io/client-go v0.19.2
	nhooyr.io/websocket v1.8.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// same version as used by rules_go to maintain compatibility with patches - see the WORKSPACE file
	golang.org/x/tools => golang.org/x/tools v0.0.0-20200823205832-c024452afbcd

	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/apiserver => k8s.io/apiserver v0.19.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.2
	k8s.io/code-generator => k8s.io/code-generator v0.19.2
	k8s.io/component-base => k8s.io/component-base v0.19.2
	k8s.io/cri-api => k8s.io/cri-api v0.19.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.2
	k8s.io/kubectl => k8s.io/kubectl v0.19.2
	k8s.io/kubelet => k8s.io/kubelet v0.19.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.19.2 // gitops-engine wants that
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.2
	k8s.io/metrics => k8s.io/metrics v0.19.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.2
)
