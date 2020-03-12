module sigs.k8s.io/cluster-api-provider-aws

go 1.14

require (
	github.com/aws/aws-sdk-go v1.29.21
	github.com/awslabs/goformation/v4 v4.4.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.3.1
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/cluster-api v0.2.10
	sigs.k8s.io/cluster-api-bootstrap-provider-kubeadm v0.1.6
	sigs.k8s.io/controller-runtime v0.5.1
	sigs.k8s.io/yaml v1.1.0
)