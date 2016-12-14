package logic

import (
	"fmt"
	"github.com/quintilesims/layer0/common/errors"
	"github.com/quintilesims/layer0/common/models"
)

type TaskLogic interface {
	CreateTask(models.CreateTaskRequest) (*models.Task, error)
	ListTasks() ([]*models.Task, error)
	GetTask(string) (*models.Task, error)
	DeleteTask(string) error
	GetTaskLogs(string, int) ([]*models.LogFile, error)
}

type L0TaskLogic struct {
	Logic
	DeployLogic
}

func NewL0TaskLogic(logic Logic, deployLogic DeployLogic) *L0TaskLogic {
	return &L0TaskLogic{
		Logic:       logic,
		DeployLogic: deployLogic,
	}
}

func (this *L0TaskLogic) ListTasks() ([]*models.Task, error) {
	tasks, err := this.Backend.ListTasks()
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if err := this.populateModel(task); err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

func (this *L0TaskLogic) GetTask(taskID string) (*models.Task, error) {
	environmentID, err := this.getEnvironmentID(taskID)
	if err != nil {
		return nil, err
	}

	task, err := this.Backend.GetTask(environmentID, taskID)
	if err != nil {
		return nil, err
	}

	if err := this.populateModel(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (this *L0TaskLogic) DeleteTask(taskID string) error {
	environmentID, err := this.getEnvironmentID(taskID)
	if err != nil {
		return err
	}

	if err := this.Backend.DeleteTask(environmentID, taskID); err != nil {
		return err
	}

	if err := this.deleteEntityTags("task", taskID); err != nil {
		return err
	}

	return nil
}

func (this *L0TaskLogic) CreateTask(req models.CreateTaskRequest) (*models.Task, error) {
	if req.EnvironmentID == "" {
		return nil, errors.Newf(errors.MissingParameter, "EnvironmentID not specified")
	}

	if req.DeployID == "" {
		return nil, errors.Newf(errors.MissingParameter, "DeployID not specified")
	}

	if req.TaskName == "" {
		return nil, errors.Newf(errors.MissingParameter, "TaskName not specified")
	}

	task, err := this.Backend.CreateTask(
		req.EnvironmentID,
		req.TaskName,
		req.DeployID,
		int(req.Copies),
		req.ContainerOverrides)
	if err != nil {
		return task, err
	}

	taskID := task.TaskID
	if err := this.upsertTagf(taskID, "task", "name", req.TaskName); err != nil {
		return task, err
	}

	environmentID := task.EnvironmentID
	if err := this.upsertTagf(taskID, "task", "environment_id", environmentID); err != nil {
		return task, err
	}

	deployID := task.DeployID
	if err := this.upsertTagf(taskID, "task", "deploy_id", deployID); err != nil {
		return task, err
	}

	if err := this.populateModel(task); err != nil {
		return task, err
	}

	return task, nil
}

func (this *L0TaskLogic) GetTaskLogs(taskID string, tail int) ([]*models.LogFile, error) {
	environmentID, err := this.getEnvironmentID(taskID)
	if err != nil {
		return nil, err
	}

	logs, err := this.Backend.GetTaskLogs(environmentID, taskID, tail)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (this *L0TaskLogic) getEnvironmentID(taskID string) (string, error) {
	tags, err := this.TagStore.SelectByQuery("task", taskID)
	if err != nil {
		return "", err
	}

	if tag := tags.WithKey("environment_id").First(); tag != nil {
		return tag.Value, nil
	}

	return "", fmt.Errorf("Failed to find Environment ID for Task %s", taskID)
}

func (this *L0TaskLogic) populateModel(model *models.Task) error {
	tags, err := this.TagStore.SelectByQuery("task", model.TaskID)
	if err != nil {
		return err
	}

	if tag := tags.WithKey("environment_id").First(); tag != nil {
		model.EnvironmentID = tag.Value
	}

	if tag := tags.WithKey("deploy_id").First(); tag != nil {
		model.DeployID = tag.Value
	}

	if tag := tags.WithKey("name").First(); tag != nil {
		model.TaskName = tag.Value
	}

	if model.EnvironmentID != "" {
		tags, err := this.TagStore.SelectByQuery("environment", model.EnvironmentID)
		if err != nil {
			return err
		}

		if tag := tags.WithKey("name").First(); tag != nil {
			model.EnvironmentName = tag.Value
		}
	}

	if model.DeployID != "" {
		tags, err := this.TagStore.SelectByQuery("deploy", model.DeployID)
		if err != nil {
			return err
		}

		if tag := tags.WithKey("name").First(); tag != nil {
			model.DeployName = tag.Value
		}

		if tag := tags.WithKey("version").First(); tag != nil {
			model.DeployVersion = tag.Value
		}
	}

	return nil
}
