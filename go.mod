module gitlab.com/gitlab-org/cluster-integration/gitlab-agent

go 1.15

require (
	github.com/argoproj/gitops-engine v0.1.3-0.20200829061729-4d6f2988b2a7
	github.com/ash2k/stager v0.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gitlab.com/gitlab-org/gitaly v1.87.1-0.20200731181045-a6091637dcb4
	gitlab.com/gitlab-org/labkit v0.0.0-20200908084045-45895e129029
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/tools v0.0.0-20200823205832-c024452afbcd
	google.golang.org/grpc v1.32.0
	google.golang.org/grpc/examples v0.0.0-20200828165940-d8ef479ab79a // indirect
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/cli-runtime v0.18.8
	k8s.io/client-go v0.18.8
	nhooyr.io/websocket v1.8.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// same version as used by rules_go to maintain compatibility with patches - see the WORKSPACE file
	golang.org/x/tools => golang.org/x/tools v0.0.0-20200823205832-c024452afbcd

	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/apiserver => k8s.io/apiserver v0.18.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/component-base => k8s.io/component-base v0.18.8
	k8s.io/cri-api => k8s.io/cri-api v0.18.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.8
	k8s.io/kubectl => k8s.io/kubectl v0.18.8
	k8s.io/kubelet => k8s.io/kubelet v0.18.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.18.8 // gitops-engine wants that
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.8
	k8s.io/metrics => k8s.io/metrics v0.18.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.8
)
