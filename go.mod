module github.com/appvia/kev

go 1.14

require (
	github.com/GoogleContainerTools/skaffold v1.15.0
	github.com/compose-spec/compose-go v0.0.0-20200907084823-057e1edc5b6f
	github.com/fsnotify/fsnotify v1.4.9
	github.com/google/go-cmp v0.5.2
	github.com/imdario/mergo v0.3.11
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/xeipuuv/gojsonschema v1.2.0
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.19.2
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.0
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200221181110-62bd5a33f707
	golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
	k8s.io/api => k8s.io/api v0.17.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.4
	k8s.io/client-go => k8s.io/client-go v0.17.4
)
