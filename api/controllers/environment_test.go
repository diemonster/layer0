package controllers

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/quintilesims/layer0/api/job"
	"github.com/quintilesims/layer0/api/job/mock_job"
	"github.com/quintilesims/layer0/api/provider/mock_provider"
	"github.com/quintilesims/layer0/common/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateEnvironment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEnvironmentProvider := mock_provider.NewMockEnvironmentProvider(ctrl)
	mockJobScheduler := mock_job.NewMockScheduler(ctrl)
	controller := NewEnvironmentController(mockEnvironmentProvider, mockJobScheduler)

	req := models.CreateEnvironmentRequest{
		EnvironmentName: "env",
		InstanceSize:    "m3.medium",
		MinClusterCount: 1,
		OperatingSystem: "linux",
		AMIID:           "ami123",
	}

	sjr := models.ScheduleJobRequest{
		JobType: job.CreateEnvironmentJob.String(),
		Request: req,
	}

	mockJobScheduler.EXPECT().
		Schedule(sjr).
		Return("jid", nil)

	c := newFireballContext(t, req, nil)
	resp, err := controller.CreateEnvironment(c)
	if err != nil {
		t.Fatal(err)
	}

	var response models.Job
	recorder := unmarshalBody(t, resp, &response)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "jid", response.JobID)
}

func TestDeleteEnvironment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEnvironmentProvider := mock_provider.NewMockEnvironmentProvider(ctrl)
	mockJobScheduler := mock_job.NewMockScheduler(ctrl)
	controller := NewEnvironmentController(mockEnvironmentProvider, mockJobScheduler)

	sjr := models.ScheduleJobRequest{
		JobType: job.DeleteEnvironmentJob.String(),
		Request: "eid",
	}

	mockJobScheduler.EXPECT().
		Schedule(sjr).
		Return("jid", nil)

	c := newFireballContext(t, nil, map[string]string{"id": "eid"})
	resp, err := controller.DeleteEnvironment(c)
	if err != nil {
		t.Fatal(err)
	}

	var response models.Job
	recorder := unmarshalBody(t, resp, &response)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "jid", response.JobID)
}

func TestGetEnvironment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEnvironmentProvider := mock_provider.NewMockEnvironmentProvider(ctrl)
	mockJobScheduler := mock_job.NewMockScheduler(ctrl)
	controller := NewEnvironmentController(mockEnvironmentProvider, mockJobScheduler)

	environmentModel := models.Environment{
		EnvironmentID:   "e1",
		EnvironmentName: "env",
		InstanceSize:    "m3.medium",
		ClusterCount:    1,
		SecurityGroupID: "sg1",
		OperatingSystem: "linux",
		AMIID:           "ami123",
		Links:           []string{"e2"},
	}

	mockEnvironmentProvider.EXPECT().
		Read("e1").
		Return(&environmentModel, nil)

	c := newFireballContext(t, nil, map[string]string{"id": "e1"})
	resp, err := controller.GetEnvironment(c)
	if err != nil {
		t.Fatal(err)
	}

	var response models.Environment
	recorder := unmarshalBody(t, resp, &response)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, environmentModel, response)
}

func TestListEnvironments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEnvironmentProvider := mock_provider.NewMockEnvironmentProvider(ctrl)
	mockJobScheduler := mock_job.NewMockScheduler(ctrl)
	controller := NewEnvironmentController(mockEnvironmentProvider, mockJobScheduler)

	environmentSummaries := []models.EnvironmentSummary{
		{
			EnvironmentID:   "e1",
			EnvironmentName: "env1",
			OperatingSystem: "linux",
		},
		{
			EnvironmentID:   "e2",
			EnvironmentName: "env2",
			OperatingSystem: "windows",
		},
	}

	mockEnvironmentProvider.EXPECT().
		List().
		Return(environmentSummaries, nil)

	c := newFireballContext(t, nil, nil)
	resp, err := controller.ListEnvironments(c)
	if err != nil {
		t.Fatal(err)
	}

	var response []models.EnvironmentSummary
	recorder := unmarshalBody(t, resp, &response)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, environmentSummaries, response)
}