module github.com/appvia/kube-devx

go 1.14

require (
	github.com/compose-spec/compose-go v0.0.0-20200629133725-59b25574fd55
	github.com/google/go-cmp v0.5.0
	github.com/imdario/mergo v0.3.9
	github.com/joho/godotenv v1.3.0
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c
	k8s.io/api v0.18.4
	k8s.io/apimachinery v0.18.4
)

replace (
	k8s.io/api => k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.5-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.18.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.4
	k8s.io/client-go => k8s.io/client-go v0.18.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.4
	k8s.io/code-generator => k8s.io/code-generator v0.18.5-rc.0
	k8s.io/component-base => k8s.io/component-base v0.18.4
	k8s.io/cri-api => k8s.io/cri-api v0.18.5-rc.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.4
	k8s.io/kubectl => k8s.io/kubectl v0.18.4
	k8s.io/kubelet => k8s.io/kubelet v0.18.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.4
	k8s.io/metrics => k8s.io/metrics v0.18.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.4
)
