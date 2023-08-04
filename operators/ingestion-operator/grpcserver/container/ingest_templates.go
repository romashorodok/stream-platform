package container

import (
	"errors"

	"github.com/romashorodok/stream-platform/operators/ingestion-operator/api/v1alpha1"
)

type IngestTemplates struct {
	templates map[string]v1alpha1.IngestTemplate
}

var instance *IngestTemplates = &IngestTemplates{
	templates: make(map[string]v1alpha1.IngestTemplate, 0),
}

func WithIngestTemplates() *IngestTemplates {
	return instance
}

func (c *IngestTemplates) AddIngestTemplate(ingestTemplate v1alpha1.IngestTemplate) error {
	c.templates[ingestTemplate.Name] = ingestTemplate

	return nil
}

func (c *IngestTemplates) Get(ingestTemplateName string) (*v1alpha1.IngestTemplate, error) {
	if ingestTemplate, ok := c.templates[ingestTemplateName]; ok == true {
		return &ingestTemplate, nil
	}
	return nil, errors.New("not found template in container")
}
