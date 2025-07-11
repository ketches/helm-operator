package installer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmOperatorInstaller provides methods to install helm-operator
type HelmOperatorInstaller struct {
	client client.Client
}

// NewInstaller creates a new HelmOperatorInstaller
func NewInstaller(client client.Client) *HelmOperatorInstaller {
	return &HelmOperatorInstaller{
		client: client,
	}
}

// Install installs the helm-operator using embedded resources
func (i *HelmOperatorInstaller) Install(ctx context.Context) error {
	// Parse and apply all resources
	for idx, resourceYAML := range resources {
		if err := i.applyResource(ctx, resourceYAML, idx); err != nil {
			return fmt.Errorf("failed to apply resource %d: %w", idx, err)
		}
	}
	return nil
}

// Uninstall removes the helm-operator resources
func (i *HelmOperatorInstaller) Uninstall(ctx context.Context) error {
	// Delete resources in reverse order
	for idx := len(resources) - 1; idx >= 0; idx-- {
		if err := i.deleteResource(ctx, resources[idx], idx); err != nil {
			// Log error but continue with other resources
			fmt.Printf("Warning: failed to delete resource %d: %v\n", idx, err)
		}
	}
	return nil
}

// applyResource applies a single YAML resource
func (i *HelmOperatorInstaller) applyResource(ctx context.Context, resourceYAML string, idx int) error {
	// Skip empty resources
	if resourceYAML == "" {
		return nil
	}

	// Parse YAML to unstructured object
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	if _, _, err := decoder.Decode([]byte(resourceYAML), nil, obj); err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Apply the resource
	if err := i.client.Create(ctx, obj); err != nil {
		// If resource already exists, try to update it
		if client.IgnoreAlreadyExists(err) == nil {
			return i.client.Update(ctx, obj)
		}
		return err
	}

	fmt.Printf("Applied resource %d: %s/%s\n", idx, obj.GetKind(), obj.GetName())
	return nil
}

// deleteResource deletes a single YAML resource
func (i *HelmOperatorInstaller) deleteResource(ctx context.Context, resourceYAML string, idx int) error {
	// Skip empty resources
	if resourceYAML == "" {
		return nil
	}

	// Parse YAML to unstructured object
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	if _, _, err := decoder.Decode([]byte(resourceYAML), nil, obj); err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Delete the resource
	if err := i.client.Delete(ctx, obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	fmt.Printf("Deleted resource %d: %s/%s\n", idx, obj.GetKind(), obj.GetName())
	return nil
}

// GetStatus returns the installation status
func (i *HelmOperatorInstaller) GetStatus(ctx context.Context) (*InstallationStatus, error) {
	status := &InstallationStatus{
		Installed: true,
		Resources: make([]ResourceStatus, 0, len(resources)),
	}

	for idx, resourceYAML := range resources {
		if resourceYAML == "" {
			continue
		}

		// Parse YAML to get resource info
		obj := &unstructured.Unstructured{}
		decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		if _, _, err := decoder.Decode([]byte(resourceYAML), nil, obj); err != nil {
			status.Resources = append(status.Resources, ResourceStatus{
				Index:  idx,
				Kind:   "Unknown",
				Name:   "Unknown",
				Exists: false,
				Error:  err.Error(),
			})
			status.Installed = false
			continue
		}

		// Check if resource exists
		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
		err := i.client.Get(ctx, client.ObjectKeyFromObject(obj), existing)

		resourceStatus := ResourceStatus{
			Index:     idx,
			Kind:      obj.GetKind(),
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Exists:    err == nil,
		}

		if err != nil && client.IgnoreNotFound(err) != nil {
			resourceStatus.Error = err.Error()
			status.Installed = false
		}

		status.Resources = append(status.Resources, resourceStatus)

		if !resourceStatus.Exists {
			status.Installed = false
		}
	}

	return status, nil
}

// InstallationStatus represents the status of the helm-operator installation
type InstallationStatus struct {
	Installed bool             `json:"installed"`
	Resources []ResourceStatus `json:"resources"`
}

// ResourceStatus represents the status of a single resource
type ResourceStatus struct {
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Exists    bool   `json:"exists"`
	Error     string `json:"error,omitempty"`
}
