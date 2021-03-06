package scheduler

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/quintilesims/layer0/api/scheduler/resource"
	"github.com/quintilesims/layer0/common/errors"
	"github.com/quintilesims/layer0/common/logutils"
	"github.com/quintilesims/layer0/common/models"
)

type EnvironmentScaler interface {
	Scale(environmentID string) (*models.ScalerRunInfo, error)
	ScheduleRun(environmentID string, delay time.Duration)
}

type L0EnvironmentScaler struct {
	consumerGetter  resource.ConsumerGetter
	providerManager resource.ProviderManager
	scheduledRuns   map[string]chan time.Duration
	logger          *logrus.Logger
}

func NewL0EnvironmentScaler(c resource.ConsumerGetter, p resource.ProviderManager) *L0EnvironmentScaler {
	return &L0EnvironmentScaler{
		consumerGetter:  c,
		providerManager: p,
		scheduledRuns:   map[string]chan time.Duration{},
		logger:          logutils.NewStandardLogger("Environment Scaler").Logger,
	}
}

func (r *L0EnvironmentScaler) ScheduleRun(environmentID string, delay time.Duration) {
	if c, ok := r.scheduledRuns[environmentID]; ok {
		c <- delay
		return
	}

	c := r.newScheduledRun(environmentID, delay)
	r.scheduledRuns[environmentID] = c
}

func (r *L0EnvironmentScaler) newScheduledRun(environmentID string, delay time.Duration) chan time.Duration {
	c := make(chan time.Duration)

	go func() {
		for shouldContinue := true; shouldContinue; {
			r.logger.Debugf("Scaling environment '%s' in %v", environmentID, delay)
			select {
			case delay = <-c:
				r.logger.Debugf("New delay set for environment '%s'", environmentID)
			case <-time.After(delay):
				delete(r.scheduledRuns, environmentID)
				close(c)
				shouldContinue = false
			}
		}

		r.logger.Debugf("Scaling environment '%s' now", environmentID)
		if _, err := r.Scale(environmentID); err != nil {
			r.logger.Errorf("There was an error scaling environment %s: %v", environmentID, err)
		}
	}()

	return c
}

func (r *L0EnvironmentScaler) Scale(environmentID string) (*models.ScalerRunInfo, error) {
	resourceProviders, err := r.providerManager.GetProviders(environmentID)
	if err != nil {
		return nil, err
	}

	resourceConsumers, err := r.consumerGetter.GetConsumers(environmentID)
	if err != nil {
		return nil, err
	}

	return RunBasicScaler(environmentID, resourceProviders, resourceConsumers, r.providerManager)
}

func RunBasicScaler(
	environmentID string,
	providers []*resource.ResourceProvider,
	consumers []resource.ResourceConsumer,
	providerManager resource.ProviderManager,
) (*models.ScalerRunInfo, error) {

	scaleBeforeRun := len(providers)
	var errs []error

	// check if we need to scale up
	for _, consumer := range consumers {
		hasRoom := false

		// first, sort by memory so we pack tasks by memory as tightly as possible
		resource.SortProvidersByMemory(providers)

		// next, place any unused providers in the back of the list
		// that way, we can can delete them if we avoid placing any tasks in them
		resource.SortProvidersByUsage(providers)

		for _, provider := range providers {
			if provider.HasResourcesFor(consumer) {
				hasRoom = true
				provider.SubtractResourcesFor(consumer)
				break
			}
		}

		if !hasRoom {
			newProvider, err := providerManager.CalculateNewProvider(environmentID)
			if err != nil {
				return nil, err
			}

			if !newProvider.HasResourcesFor(consumer) {
				text := fmt.Sprintf("Resource '%s' cannot fit into an empty provider!", consumer.ID)
				text += "\nThe instance size in your environment is too small to run this resource."
				text += "\nPlease increase the instance size for your environment"
				err := fmt.Errorf(text)
				errs = append(errs, err)
				continue
			}

			newProvider.SubtractResourcesFor(consumer)
			providers = append(providers, newProvider)
		}
	}

	// check if we need to scale down
	unusedProviders := []*resource.ResourceProvider{}
	for i := 0; i < len(providers); i++ {
		if !providers[i].IsInUse() {
			unusedProviders = append(unusedProviders, providers[i])
		}
	}

	desiredScale := len(providers) - len(unusedProviders)
	actualScale, err := providerManager.ScaleTo(environmentID, desiredScale, unusedProviders)
	if err != nil {
		errs = append(errs, err)
	}

	info := &models.ScalerRunInfo{
		EnvironmentID:           environmentID,
		PendingResources:        resourceConsumerModels(consumers),
		ResourceProviders:       resourceProviderModels(providers),
		ScaleBeforeRun:          scaleBeforeRun,
		DesiredScaleAfterRun:    desiredScale,
		ActualScaleAfterRun:     actualScale,
		UnusedResourceProviders: len(unusedProviders),
	}

	return info, errors.MultiError(errs)
}

func resourceConsumerModels(consumers []resource.ResourceConsumer) []models.ResourceConsumer {
	models := make([]models.ResourceConsumer, len(consumers))
	for i := 0; i < len(consumers); i++ {
		models[i] = consumers[i].ToModel()
	}

	return models
}

func resourceProviderModels(providers []*resource.ResourceProvider) []models.ResourceProvider {
	models := make([]models.ResourceProvider, len(providers))
	for i := 0; i < len(providers); i++ {
		models[i] = providers[i].ToModel()
	}

	return models
}
