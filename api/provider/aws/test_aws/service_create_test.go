package test_aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/golang/mock/gomock"
	provider "github.com/quintilesims/layer0/api/provider/aws"
	"github.com/quintilesims/layer0/api/tag"
	awsc "github.com/quintilesims/layer0/common/aws"
	"github.com/quintilesims/layer0/common/config"
	"github.com/quintilesims/layer0/common/config/mock_config"
	"github.com/quintilesims/layer0/common/models"
	"github.com/stretchr/testify/assert"
)

func TestServiceCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAWS := awsc.NewMockClient(ctrl)
	tagStore := tag.NewMemoryStore()
	mockConfig := mock_config.NewMockAPIConfig(ctrl)

	mockConfig.EXPECT().Instance().Return("test").AnyTimes()

	tags := models.Tags{
		{
			EntityID:   "dpl_id",
			EntityType: "deploy",
			Key:        "arn",
			Value:      "dpl_arn",
		},
	}

	for _, tag := range tags {
		if err := tagStore.Insert(tag); err != nil {
			t.Fatal(err)
		}
	}

	defer provider.SetEntityIDGenerator("svc_id")()

	loadBalancerInput := &elb.DescribeLoadBalancersInput{}
	loadBalancerInput.SetLoadBalancerNames([]*string{aws.String("l0-test-lb_id")})
	loadBalancerInput.SetPageSize(1)

	listener := &elb.Listener{}
	listener.SetInstancePort(config.DefaultLoadBalancerPort().HostPort)

	listenerDescription := &elb.ListenerDescription{}
	listenerDescription.SetListener(listener)

	listenerDescriptions := []*elb.ListenerDescription{
		listenerDescription,
	}

	loadBalancerDescription := &elb.LoadBalancerDescription{}
	loadBalancerDescription.SetListenerDescriptions(listenerDescriptions)
	loadBalancerDescription.SetLoadBalancerName("l0-test-lb_id")

	loadBalancerDescriptions := []*elb.LoadBalancerDescription{
		loadBalancerDescription,
	}

	loadBalancerOutput := &elb.DescribeLoadBalancersOutput{}
	loadBalancerOutput.SetLoadBalancerDescriptions(loadBalancerDescriptions)

	mockAWS.ELB.EXPECT().
		DescribeLoadBalancers(loadBalancerInput).
		Return(loadBalancerOutput, nil)

	taskDefinitionInput := &ecs.DescribeTaskDefinitionInput{}
	taskDefinitionInput.SetTaskDefinition("dpl_arn")

	portMapping := &ecs.PortMapping{}
	portMapping.SetContainerPort(int64(80))

	portMappings := []*ecs.PortMapping{
		portMapping,
	}

	containerDefinition := &ecs.ContainerDefinition{}
	containerDefinition.SetName("ctn_name")
	containerDefinition.SetPortMappings(portMappings)

	containerDefinitions := []*ecs.ContainerDefinition{
		containerDefinition,
	}

	taskDefinition := &ecs.TaskDefinition{
		ContainerDefinitions: containerDefinitions,
	}

	taskDefinitionOutput := &ecs.DescribeTaskDefinitionOutput{}
	taskDefinitionOutput.SetTaskDefinition(taskDefinition)

	mockAWS.ECS.EXPECT().
		DescribeTaskDefinition(taskDefinitionInput).
		Return(taskDefinitionOutput, nil)

	createServiceInput := &ecs.CreateServiceInput{}
	createServiceInput.SetCluster("l0-test-env_id")
	createServiceInput.SetDesiredCount(1)
	createServiceInput.SetServiceName("l0-test-svc_id")
	createServiceInput.SetTaskDefinition("dpl_arn")

	loadBalancer := &ecs.LoadBalancer{}
	loadBalancer.SetContainerName("ctn_name")
	loadBalancer.SetContainerPort(80)
	loadBalancer.SetLoadBalancerName("l0-test-lb_id")

	loadBalancers := []*ecs.LoadBalancer{loadBalancer}

	createServiceInput.SetLoadBalancers(loadBalancers)
	createServiceInput.SetRole("l0-test-lb_id-lb")

	mockAWS.ECS.EXPECT().
		CreateService(createServiceInput).
		Return(&ecs.CreateServiceOutput{}, nil)

	req := models.CreateServiceRequest{
		DeployID:       "dpl_id",
		EnvironmentID:  "env_id",
		LoadBalancerID: "lb_id",
		ServiceName:    "svc_name",
	}

	target := provider.NewServiceProvider(mockAWS.Client(), tagStore, mockConfig)
	result, err := target.Create(req)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, result, "svc_id")

	expectedTags := models.Tags{
		{
			EntityID:   "svc_id",
			EntityType: "service",
			Key:        "name",
			Value:      "svc_name",
		},
		{
			EntityID:   "svc_id",
			EntityType: "service",
			Key:        "environment_id",
			Value:      "env_id",
		},
		{
			EntityID:   "svc_id",
			EntityType: "service",
			Key:        "load_balancer_id",
			Value:      "lb_id",
		},
	}

	for _, tag := range expectedTags {
		assert.Contains(t, tagStore.Tags(), tag)
	}
}

func TestServiceCreate_defaults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAWS := awsc.NewMockClient(ctrl)
	tagStore := tag.NewMemoryStore()
	mockConfig := mock_config.NewMockAPIConfig(ctrl)

	mockConfig.EXPECT().Instance().Return("test").AnyTimes()

	tags := models.Tags{
		{
			EntityID:   "dpl_id",
			EntityType: "deploy",
			Key:        "arn",
			Value:      "dpl_arn",
		},
	}

	for _, tag := range tags {
		if err := tagStore.Insert(tag); err != nil {
			t.Fatal(err)
		}
	}

	defer provider.SetEntityIDGenerator("svc_id")()

	createServiceInput := &ecs.CreateServiceInput{}
	createServiceInput.SetCluster("l0-test-env_id")
	createServiceInput.SetDesiredCount(1)
	createServiceInput.SetServiceName("l0-test-svc_id")
	createServiceInput.SetTaskDefinition("dpl_arn")

	mockAWS.ECS.EXPECT().
		CreateService(createServiceInput).
		Return(&ecs.CreateServiceOutput{}, nil)

	req := models.CreateServiceRequest{
		DeployID:      "dpl_id",
		EnvironmentID: "env_id",
		ServiceName:   "svc_name",
	}

	target := provider.NewServiceProvider(mockAWS.Client(), tagStore, mockConfig)
	if _, err := target.Create(req); err != nil {
		t.Fatal(err)
	}
}

func TestServiceCreateRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAWS := awsc.NewMockClient(ctrl)
	tagStore := tag.NewMemoryStore()
	mockConfig := mock_config.NewMockAPIConfig(ctrl)

	mockConfig.EXPECT().Instance().Return("test").AnyTimes()

	tags := models.Tags{
		{
			EntityID:   "dpl_id",
			EntityType: "deploy",
			Key:        "arn",
			Value:      "dpl_arn",
		},
	}

	for _, tag := range tags {
		if err := tagStore.Insert(tag); err != nil {
			t.Fatal(err)
		}
	}

	defer provider.SetEntityIDGenerator("svc_id")()

	gomock.InOrder(
		mockAWS.ECS.EXPECT().
			CreateService(gomock.Any()).
			Return(nil, fmt.Errorf("Unable to assume role and validate ports")),
		mockAWS.ECS.EXPECT().
			CreateService(gomock.Any()).
			Return(&ecs.CreateServiceOutput{}, nil),
	)

	req := models.CreateServiceRequest{
		DeployID:      "dpl_id",
		EnvironmentID: "env_id",
		ServiceName:   "svc_name",
	}

	target := provider.NewServiceProvider(mockAWS.Client(), tagStore, mockConfig)
	if _, err := target.Create(req); err != nil {
		t.Fatal(err)
	}
}