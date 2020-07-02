module gitlab.com/gitlab-org/cluster-integration/gitlab-agent

go 1.14

require (
	github.com/argoproj/gitops-engine v0.1.3-0.20200623172753-7d3da9f16e35
	github.com/ash2k/stager v0.1.0
	github.com/golang/mock v1.4.4-0.20200612212805-d9ac6780152f
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.4.1
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	gitlab.com/gitlab-org/gitaly v1.87.1-0.20200519214319-382ead9c7ef3
	gitlab.com/gitlab-org/labkit v0.0.0-20200622172558-49c073024c24
	golang.org/x/net v0.0.0-20200506145744-7e3656a0809f // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200509044756-6aff5f38e54f // indirect
	golang.org/x/tools v0.0.0-20200623204733-f8e0ea3a3a8f
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.24.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/apiextensions-apiserver v0.17.6 // indirect
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190816222228-6d55c1b1f1ca+incompatible
	k8s.io/kube-aggregator v0.17.6 // indirect
	k8s.io/kubectl v0.17.6 // indirect
	nhooyr.io/websocket v1.8.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/apiserver => k8s.io/apiserver v0.17.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
	k8s.io/component-base => k8s.io/component-base v0.17.6
	k8s.io/cri-api => k8s.io/cri-api v0.17.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.6
	k8s.io/kubectl => k8s.io/kubectl v0.17.6
	k8s.io/kubelet => k8s.io/kubelet v0.17.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.17.6 // gitops-engine wants that
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.6
	k8s.io/metrics => k8s.io/metrics v0.17.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.6
)
