package ecsbackend

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	aws_ecs "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"
	"gitlab.imshealth.com/xfra/layer0/api/backend/ecs/id"
	"gitlab.imshealth.com/xfra/layer0/api/backend/ecs/mock_ecsbackend"
	"gitlab.imshealth.com/xfra/layer0/api/backend/mock_backend"
	"gitlab.imshealth.com/xfra/layer0/common/aws/cloudwatchlogs"
	"gitlab.imshealth.com/xfra/layer0/common/aws/cloudwatchlogs/mock_cloudwatchlogs"
	"gitlab.imshealth.com/xfra/layer0/common/aws/ecs"
	"gitlab.imshealth.com/xfra/layer0/common/aws/ecs/mock_ecs"
	"gitlab.imshealth.com/xfra/layer0/common/models"
	"gitlab.imshealth.com/xfra/layer0/common/testutils"
	"testing"
)

type MockECSTaskManager struct {
	ECS            *mock_ecs.MockProvider
	CloudWatchLogs *mock_cloudwatchlogs.MockProvider
	Backend        *mock_backend.MockBackend
	ClusterScaler  *mock_ecsbackend.MockClusterScaler
	Scheduler      *mock_ecsbackend.MockTaskScheduler
}

func NewMockECSTaskManager(ctrl *gomock.Controller) *MockECSTaskManager {
	return &MockECSTaskManager{
		ECS:            mock_ecs.NewMockProvider(ctrl),
		CloudWatchLogs: mock_cloudwatchlogs.NewMockProvider(ctrl),
		Backend:        mock_backend.NewMockBackend(ctrl),
		ClusterScaler:  mock_ecsbackend.NewMockClusterScaler(ctrl),
		Scheduler:      mock_ecsbackend.NewMockTaskScheduler(ctrl),
	}
}

func (this *MockECSTaskManager) Task() *ECSTaskManager {
	taskManager := NewECSTaskManager(this.ECS, this.CloudWatchLogs, this.Backend, this.ClusterScaler)
	taskManager.Scheduler = this.Scheduler
	return taskManager
}

func TestGetTask(t *testing.T) {
	testCases := []testutils.TestCase{
		testutils.TestCase{
			Name: "Should call ecs.ListTasks, scheduler.GetTask with proper params",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				taskID := id.L0TaskID("tskid").ECSTaskID()

				mockTask.ECS.EXPECT().
					ListTasks(environmentID.String(), nil, gomock.Any(), stringp(taskID.String()), nil).
					Return(nil, nil).
					Times(3)

				mockTask.Scheduler.EXPECT().
					GetTask(taskID).
					Return(nil)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.GetTask("envid", "tskid")
			},
		},
		testutils.TestCase{
			Name: "Should return layer0-formatted ids",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				clusterARN := fmt.Sprintf("arn:aws:ecs:region:aws_account_id:cluster/%s", environmentID.String())
				taskID := id.L0TaskID("tskid").ECSTaskID()
				deployID := id.L0DeployID("dply.1").ECSDeployID()
				deployARN := fmt.Sprintf("arn:aws:ecs:region:aws_account_id:cluster/%s", deployID.String())

				mockTask.ECS.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil).
					Times(3)

				mockTask.Scheduler.EXPECT().
					GetTask(gomock.Any()).
					Return([]*ecs.Task{ecs.NewTask(clusterARN, taskID.String(), deployARN)})

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)

				task, err := manager.GetTask("envid", "tskid")
				if err != nil {
					reporter.Fatal(err)
				}

				reporter.AssertEqual(task.TaskID, "tskid")
				reporter.AssertEqual(task.EnvironmentID, "envid")
				reporter.AssertEqual(task.DeployID, "dply.1")
			},
		},
	}

	testutils.RunTests(t, testCases)
}

func TestListTasks(t *testing.T) {
	testCases := []testutils.TestCase{
		testutils.TestCase{
			Name: "Should use proper params in dependent calls",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid")
				environments := []*models.Environment{
					&models.Environment{
						EnvironmentID: environmentID.String(),
					},
				}

				mockTask.Backend.EXPECT().
					ListEnvironments().
					Return(environments, nil)

				mockTask.ECS.EXPECT().
					ListTasks(environmentID.ECSEnvironmentID().String(), nil, gomock.Any(), nil, nil).
					Return(nil, nil).
					Times(3)

				mockTask.Scheduler.EXPECT().
					ListTasks().
					Return(nil)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.ListTasks()
			},
		},
		testutils.TestCase{
			Name: "Should propagate ecs.ListTasks error",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid")
				environments := []*models.Environment{
					&models.Environment{
						EnvironmentID: environmentID.String(),
					},
				}

				mockTask.Backend.EXPECT().
					ListEnvironments().
					Return(environments, nil)

				mockTask.ECS.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("some error"))

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)

				if _, err := manager.ListTasks(); err == nil {
					reporter.Fatalf("Error was nil!")
				}
			},
		},
	}

	testutils.RunTests(t, testCases)
}

func TestDeleteTask(t *testing.T) {
	testCases := []testutils.TestCase{
		testutils.TestCase{
			Name: "Should use proper params in dependent calls",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				taskID := id.L0TaskID("tskid").ECSTaskID()

				mockTask.ECS.EXPECT().
					ListTasks(environmentID.String(), nil, gomock.Any(), stringp(taskID.String()), nil).
					Return(nil, nil).
					Times(3)

				mockTask.Scheduler.EXPECT().
					DeleteTask(taskID)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.DeleteTask("envid", "tskid")
			},
		},
		testutils.TestCase{
			Name: "Should propagate ecs.ListTasks error",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				mockTask.ECS.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("some error"))

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)

				if err := manager.DeleteTask("envid", "tskid"); err == nil {
					reporter.Fatalf("Error was nil!")
				}
			},
		},
	}

	testutils.RunTests(t, testCases)
}

func TestCreateTask(t *testing.T) {
	defer id.StubIDGeneration("tskid")()

	testCases := []testutils.TestCase{
		testutils.TestCase{
			Name: "Should use proper params in aws calls",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				deployID := id.L0DeployID("dplyid.1").ECSDeployID()
				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				taskID := id.L0TaskID("tskid").ECSTaskID()
				logGroupID := taskID.LogGroupID(environmentID)

				mockTask.CloudWatchLogs.EXPECT().
					CreateLogGroup(logGroupID).
					Return(nil)

				task := &ecs.TaskDefinition{
					&aws_ecs.TaskDefinition{
						Revision: int64p(1),
						Family:   stringp(deployID.FamilyName()),
					},
				}

				mockTask.ECS.EXPECT().
					DescribeTaskDefinition(deployID.TaskDefinition()).
					Return(task, nil).
					AnyTimes()

				renderedDeployID := id.L0DeployID("renderedid.2").ECSDeployID()
				mockTask.ClusterScaler.EXPECT().
					TriggerScalingAlgorithm(environmentID, &renderedDeployID, 1).
					Return(0, false, nil)

				mockTask.ECS.EXPECT().RunTask(
					environmentID.String(),
					renderedDeployID.TaskDefinition(),
					int64(1),
					stringp(taskID.String()),
					[]*ecs.ContainerOverride{},
				).Return(nil, nil)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.CreateTask("envid", "tsk_name", "dplyid.1", 1, nil, false, testCreateDeploy)
			},
		},
		testutils.TestCase{
			Name: "Should not create cloudwatch logs group if disableLogging is true",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				mockTask.ClusterScaler.EXPECT().
					TriggerScalingAlgorithm(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(0, false, nil)

				mockTask.ECS.EXPECT().RunTask(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, nil)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.CreateTask("envid", "tsk_name", "dplyid.1", 1, nil, true, testCreateDeploy)
			},
		},
		testutils.TestCase{
			Name: "Should add task to scheduler when cluster capacity is low",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				deployID := id.L0DeployID("dplyid.1").ECSDeployID()
				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				taskID := id.L0TaskID("tskid").ECSTaskID()

				mockTask.ClusterScaler.EXPECT().
					TriggerScalingAlgorithm(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(0, false, nil)

				err := awserr.New("", "No Container Instances were found in your cluster", nil)
				mockTask.ECS.EXPECT().RunTask(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, err)

				mockTask.Scheduler.EXPECT().
					AddTask(taskID, deployID, environmentID, 1, nil)

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.CreateTask("envid", "tsk_name", "dplyid.1", 1, nil, true, testCreateDeploy)
			},
		},
	}

	testutils.RunTests(t, testCases)
}

func TestGetTaskLogs(t *testing.T) {
	tmp := GetLogs
	defer func() { GetLogs = tmp }()

	testCases := []testutils.TestCase{
		testutils.TestCase{
			Name: "Should call GetLogs with proper params",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				environmentID := id.L0EnvironmentID("envid").ECSEnvironmentID()
				taskID := id.L0TaskID("tskid").ECSTaskID()

				// ensure we actually call GetLogs
				recorder := testutils.NewRecorder(ctrl)
				recorder.EXPECT().Call("")

				GetLogs = func(cloudWatchLogs cloudwatchlogs.Provider, logGroupID string, tail int) ([]*models.LogFile, error) {
					recorder.Call("")

					reporter.AssertEqual(logGroupID, taskID.LogGroupID(environmentID))
					reporter.AssertEqual(tail, 100)

					return nil, nil
				}

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)
				manager.GetTaskLogs("envid", "tskid", 100)
			},
		},
		testutils.TestCase{
			Name: "Should propagate GetLogs error",
			Setup: func(reporter *testutils.Reporter, ctrl *gomock.Controller) interface{} {
				mockTask := NewMockECSTaskManager(ctrl)

				// ensure we actually call GetLogs
				recorder := testutils.NewRecorder(ctrl)
				recorder.EXPECT().Call("")

				GetLogs = func(cloudWatchLogs cloudwatchlogs.Provider, logGroupID string, tail int) ([]*models.LogFile, error) {
					recorder.Call("")

					return nil, fmt.Errorf("some error")
				}

				return mockTask.Task()
			},
			Run: func(reporter *testutils.Reporter, target interface{}) {
				manager := target.(*ECSTaskManager)

				if _, err := manager.GetTaskLogs("envid", "tskid", 100); err == nil {
					reporter.Fatalf("Error was nil!")
				}
			},
		},
	}

	testutils.RunTests(t, testCases)
}