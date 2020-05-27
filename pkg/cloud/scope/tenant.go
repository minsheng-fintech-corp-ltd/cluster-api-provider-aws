package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"k8s.io/klog/klogr"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-aws/version"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type TenantConfigScope struct {
	AWSClients
	logr.Logger
	patchHelper  *patch.Helper
	ClusterScope *ClusterScope
	TenantConfig *infrav1.TenantConfig
}

type TenantConfigParams struct {
	AWSClients
	Logger       logr.Logger
	Client       client.Client
	TenantConfig *infrav1.TenantConfig
	ClusterScope *ClusterScope
}

func NewTenantConfigScope(params TenantConfigParams) (*TenantConfigScope, error) {
	if params.TenantConfig == nil {
		return nil, errors.New("failed to generate new scope from nil TenantConfig")
	}
	if params.ClusterScope == nil {
		return nil, errors.New("failed to generate new scope from nil ClusterScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.TenantConfig, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	session, err := sessionForRegion(params.TenantConfig.Spec.Region, params.TenantConfig.Spec.RoleArn)
	if err != nil {
		return nil, errors.Errorf("failed to create aws session: %v", err)
	}

	userAgentHandler := request.NamedHandler{
		Name: "capa/user-agent",
		Fn:   request.MakeAddToUserAgentHandler("aws.cluster.x-k8s.io", version.Get().String()),
	}

	if params.AWSClients.EC2 == nil {
		ec2Client := ec2.New(session)
		ec2Client.Handlers.Build.PushFrontNamed(userAgentHandler)
		ec2Client.Handlers.Complete.PushBack(recordAWSPermissionsIssue(params.TenantConfig))
		params.AWSClients.EC2 = ec2Client
	}

	if params.AWSClients.ELB == nil {
		elbClient := elb.New(session)
		elbClient.Handlers.Build.PushFrontNamed(userAgentHandler)
		elbClient.Handlers.Complete.PushBack(recordAWSPermissionsIssue(params.TenantConfig))
		params.AWSClients.ELB = elbClient
	}

	if params.AWSClients.ResourceTagging == nil {
		resourceTagging := resourcegroupstaggingapi.New(session)
		resourceTagging.Handlers.Build.PushFrontNamed(userAgentHandler)
		resourceTagging.Handlers.Complete.PushBack(recordAWSPermissionsIssue(params.TenantConfig))
		params.AWSClients.ResourceTagging = resourceTagging
	}

	if params.AWSClients.SecretsManager == nil {
		sClient := secretsmanager.New(session)
		sClient.Handlers.Complete.PushBack(recordAWSPermissionsIssue(params.TenantConfig))
		params.AWSClients.SecretsManager = sClient
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &TenantConfigScope{
		Logger:       params.Logger,
		patchHelper:  helper,
		ClusterScope: params.ClusterScope,
		TenantConfig: params.TenantConfig,
	}, nil
}

// Network returns the cluster network object.
func (s *TenantConfigScope) Network() *infrav1.Network {
	return &s.TenantConfig.Status.Network
}

// VPC returns the cluster VPC.
func (s *TenantConfigScope) VPC() *infrav1.VPCSpec {
	return &s.TenantConfig.Spec.NetworkSpec.VPC
}

// Subnets returns the cluster subnets.
func (s *TenantConfigScope) Subnets() infrav1.Subnets {
	return s.TenantConfig.Spec.NetworkSpec.Subnets
}

// SecurityGroups returns the cluster security groups as a map, it creates the map if empty.
func (s *TenantConfigScope) SecurityGroups() map[infrav1.SecurityGroupRole]infrav1.SecurityGroup {
	return s.TenantConfig.Status.Network.SecurityGroups
}
func (s *TenantConfigScope) AdditionalTags() infrav1.Tags {
	tags := s.ClusterScope.AdditionalTags()
	for key, value := range s.AdditionalTags() {
		tags[key] = value
	}
	return tags
}

// Region returns the cluster region.
func (s *TenantConfigScope) Region() string {
	return s.TenantConfig.Spec.Region
}

// PatchObject persists the cluster configuration and status.
func (s *TenantConfigScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.TenantConfig)
}

// Name return the name of cluster
func (s *TenantConfigScope) Name() string {
	return s.TenantConfig.Name
}

// Close closes the current scope persisting the cluster configuration and status.

func (s *TenantConfigScope) Close() error {
	return multierr.Combine(s.ClusterScope.Close(), s.PatchObject())
}

// ControlPlaneLoadBalancer returns the AWSLoadBalancerSpec
func (s *TenantConfigScope) ControlPlaneLoadBalancer() *infrav1.AWSLoadBalancerSpec {
	return s.TenantConfig.Spec.ControlPlaneLoadBalancer
}

// ControlPlaneLoadBalancerScheme returns the Classic ELB scheme (public or internal facing)
func (s *TenantConfigScope) ControlPlaneLoadBalancerScheme() infrav1.ClassicELBScheme {
	if s.ControlPlaneLoadBalancer() != nil && s.ControlPlaneLoadBalancer().Scheme != nil {
		return *s.ControlPlaneLoadBalancer().Scheme
	}
	return infrav1.ClassicELBSchemeInternetFacing
}
