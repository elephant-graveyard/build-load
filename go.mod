module github.com/homeport/build-load

go 1.15

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gonvenience/bunt v1.3.1
	github.com/gonvenience/neat v1.3.5
	github.com/gonvenience/text v1.0.6
	github.com/gonvenience/wrap v1.1.0
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/onsi/ginkgo v1.15.1
	github.com/onsi/gomega v1.11.0
	github.com/shipwright-io/build v0.3.1-0.20210305111301-3e3bf18672a3
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/tektoncd/pipeline v0.20.1
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.19.8
	k8s.io/apimachinery v0.19.8
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
)

replace k8s.io/client-go => k8s.io/client-go v0.19.8
