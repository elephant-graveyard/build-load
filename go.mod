module github.com/homeport/build-load

go 1.15

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/cli v0.0.0-20200210162036-a4bedce16568 // indirect
	github.com/gonvenience/bunt v1.1.4
	github.com/gonvenience/neat v1.3.5
	github.com/gonvenience/text v1.0.6
	github.com/gonvenience/wrap v1.1.0
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/shipwright-io/build v0.3.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/tektoncd/pipeline v0.20.1
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	knative.dev/test-infra v0.0.0-20200519015156-82551620b0a9 // indirect
)

replace (
    github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	k8s.io/client-go => k8s.io/client-go v0.18.10 // Required by prometheus-operator 
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // resolve `case-insensitive import collision` for gnostic/openapiv2 package 
)	
