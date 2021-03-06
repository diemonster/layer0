package logic

import (
	"time"

	"github.com/quintilesims/layer0/common/db/tag_store"
	"github.com/quintilesims/layer0/common/errors"
	"github.com/quintilesims/layer0/common/logutils"
	"github.com/quintilesims/layer0/common/waitutils"
)

const (
	taskJanitorSleepDuration = time.Minute * 10
)

var tagLogger = logutils.NewStackTraceLogger("Tags Janitor")

type TagJanitor struct {
	TaskLogic TaskLogic
	TagStore  tag_store.TagStore
	Clock     waitutils.Clock
}

func NewTagJanitor(taskLogic TaskLogic, tagStore tag_store.TagStore) *TagJanitor {
	return &TagJanitor{
		TaskLogic: taskLogic,
		TagStore:  tagStore,
		Clock:     waitutils.RealClock{},
	}
}

func (t *TagJanitor) Run() {
	go func() {
		for {
			tagLogger.Info("Starting cleanup")
			t.pulse()
			tagLogger.Infof("Finished cleanup")
			t.Clock.Sleep(taskJanitorSleepDuration)
		}
	}()
}

func (t *TagJanitor) pulse() error {
	tasks, err := t.TaskLogic.ListTasks()
	if err != nil {
		tagLogger.Errorf("Failed to list tasks: %v", err)
		return err
	}

	taskExists := func(id string) bool {
		for _, t := range tasks {
			if t.TaskID == id {
				return true
			}
		}

		return false
	}

	tags, err := t.TagStore.SelectByType("task")
	if err != nil {
		tagLogger.Errorln("Could not query for tag store for task entity type - ", err.Error())
	}

	errs := []error{}
	for _, tag := range tags {
		if !taskExists(tag.EntityID) {
			if err := t.TagStore.Delete(tag.EntityType, tag.EntityID, tag.Key); err != nil {
				tagLogger.Errorf("Could not delete tag (%#v) -  %s\n", tag, err.Error())
				continue
			}

			tagLogger.Infof("Tag for task (%s) has been deleted\n", tag.EntityID)
		}
	}

	return errors.MultiError(errs)
}
