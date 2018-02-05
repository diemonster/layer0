package aws

import (
	"testing"

	"github.com/quintilesims/layer0/api/tag"
	"github.com/quintilesims/layer0/common/models"
	"github.com/stretchr/testify/assert"
)

func TestLoadBalancer_makeLoadBalancerModel(t *testing.T) {
	tagStore := tag.NewMemoryStore()
	loadBalancer := NewLoadBalancerProvider(nil, tagStore, nil)

	tags := models.Tags{
		{
			EntityID:   "lid",
			EntityType: "load_balancer",
			Key:        "name",
			Value:      "lname",
		},
		{
			EntityID:   "lid",
			EntityType: "load_balancer",
			Key:        "environment_id",
			Value:      "eid",
		},
		{
			EntityID:   "eid",
			EntityType: "environment",
			Key:        "name",
			Value:      "ename",
		},
		{
			EntityID:   "sid",
			EntityType: "service",
			Key:        "name",
			Value:      "sname",
		},
		{
			EntityID:   "sid",
			EntityType: "service",
			Key:        "load_balancer_id",
			Value:      "lid",
		},
		{
			EntityID:   "lid",
			EntityType: "service",
			Key:        "name",
			Value:      "badname",
		},
		{
			EntityID:   "bid",
			EntityType: "load_balancer",
			Key:        "environment_id",
			Value:      "badid",
		},
	}

	for _, tag := range tags {
		if err := tagStore.Insert(tag); err != nil {
			t.Fatal(err)
		}
	}

	result, err := loadBalancer.makeLoadBalancerModel("lid")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "lname", result.LoadBalancerName)
	assert.Equal(t, "eid", result.EnvironmentID)
	assert.Equal(t, "ename", result.EnvironmentName)
	assert.Equal(t, "sid", result.ServiceID)
	assert.Equal(t, "sname", result.ServiceName)
}