module sigs.k8s.io/cluster-api-provider-aws/test/e2e

go 1.12

require (
	github.com/aws/aws-sdk-go v1.29.21
	github.com/coreos/etcd v3.3.15+incompatible // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/vmware-tanzu/sonobuoy v0.16.4
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.1-0.20190704100234-640d9f240853+incompatible
	sigs.k8s.io/cluster-api v0.2.10
	sigs.k8s.io/cluster-api-bootstrap-provider-kubeadm v0.1.6
	sigs.k8s.io/cluster-api-provider-aws v0.4.4
	sigs.k8s.io/controller-runtime v0.5.1
	sigs.k8s.io/testing_frameworks v0.1.2 // indirect
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	sigs.k8s.io/cluster-api-provider-aws => ../../
)
