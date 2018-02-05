package client

import (
	"fmt"

	"github.com/quintilesims/layer0/common/models"
)

func (c *APIClient) CreateEnvironment(req models.CreateEnvironmentRequest) (string, error) {
	var resp models.CreateEntityResponse
	if err := c.client.Post("/environment", req, &resp); err != nil {
		return "", err
	}

	return resp.EntityID, nil
}

func (c *APIClient) DeleteEnvironment(environmentID string) error {
	path := fmt.Sprintf("/environment/%s", environmentID)
	if err := c.client.Delete(path, nil, nil); err != nil {
		return err
	}

	return nil
}

func (c *APIClient) ListEnvironments() ([]models.EnvironmentSummary, error) {
	var environments []models.EnvironmentSummary
	if err := c.client.Get("/environment", &environments); err != nil {
		return nil, err
	}

	return environments, nil
}

func (c *APIClient) ReadEnvironment(environmentID string) (*models.Environment, error) {
	var environment *models.Environment
	path := fmt.Sprintf("/environment/%s", environmentID)
	if err := c.client.Get(path, &environment); err != nil {
		return nil, err
	}

	return environment, nil
}

func (c *APIClient) UpdateEnvironment(environmentID string, req models.UpdateEnvironmentRequest) error {
	path := fmt.Sprintf("/environment/%s", environmentID)
	if err := c.client.Patch(path, req, nil); err != nil {
		return err
	}

	return nil
}