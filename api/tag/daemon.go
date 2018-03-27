package tag

import (
	"log"

	"github.com/quintilesims/layer0/api/provider"
)

func NewDaemonFN(tagStore Store, taskProvider provider.TaskProvider) func() error {
	return func() error {
		tasks, err := taskProvider.List()
		if err != nil {
			return err
		}

		tags, err := tagStore.SelectByType("task")
		if err != nil {
			return err
		}

		m := make(map[string]bool, len(tasks))
		for _, task := range tasks {
			m[task.TaskID] = true
		}

		for _, tag := range tags {
			if !m[tag.EntityID] {
				log.Printf("[DEBUG] [TagDaemon] Deleting tag %#v", tag)
				if err := tagStore.Delete(tag.EntityType, tag.EntityID, tag.Key); err != nil {
					return err
				}
			}
		}

		return nil
	}
}