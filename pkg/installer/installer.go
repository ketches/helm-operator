package installer

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultNamespace is the namespace where helm-operator is installed.
// It is not deleted on Uninstall so that other resources in the namespace are preserved.
const DefaultNamespace = "ketches"

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

// ensureNamespace creates the DefaultNamespace if it does not exist.
func (i *HelmOperatorInstaller) ensureNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{}
	err := i.client.Get(ctx, client.ObjectKey{Name: DefaultNamespace}, ns)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace %s: %w", DefaultNamespace, err)
	}
	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultNamespace,
		},
	}
	if err := i.client.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", DefaultNamespace, err)
	}
	fmt.Printf("Created namespace: %s\n", DefaultNamespace)
	return nil
}

// Install installs the helm-operator using embedded resources
func (i *HelmOperatorInstaller) Install(ctx context.Context) error {
	// Ensure ketches namespace exists before applying any resource
	if err := i.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("ensure namespace: %w", err)
	}
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

	// Split multi-document YAML
	documents := strings.Split(resourceYAML, "\n---\n")

	for _, doc := range documents {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		// Parse YAML to unstructured object
		obj := &unstructured.Unstructured{}
		decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		if _, _, err := decoder.Decode([]byte(doc), nil, obj); err != nil {
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		// Apply the resource
		if err := i.client.Create(ctx, obj); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			// Resource exists: get current version and update (Update requires resourceVersion)
			existing := &unstructured.Unstructured{}
			existing.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
			if getErr := i.client.Get(ctx, client.ObjectKeyFromObject(obj), existing); getErr != nil {
				return fmt.Errorf("failed to get existing %s %s for update: %w", obj.GetKind(), obj.GetName(), getErr)
			}
			obj.SetResourceVersion(existing.GetResourceVersion())
			if err := i.client.Update(ctx, obj); err != nil {
				return fmt.Errorf("failed to update %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
			fmt.Printf("Updated existing resource %d: %s/%s\n", idx, obj.GetKind(), obj.GetName())
			continue
		}
		fmt.Printf("Applied resource %d: %s/%s\n", idx, obj.GetKind(), obj.GetName())
	}
	return nil
}

// deleteResource deletes a single YAML resource
func (i *HelmOperatorInstaller) deleteResource(ctx context.Context, resourceYAML string, idx int) error {
	// Skip empty resources
	if resourceYAML == "" {
		return nil
	}

	// Split multi-document YAML
	documents := strings.Split(resourceYAML, "\n---\n")

	for _, doc := range documents {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		// Parse YAML to unstructured object
		obj := &unstructured.Unstructured{}
		decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		if _, _, err := decoder.Decode([]byte(doc), nil, obj); err != nil {
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		// Do not delete the ketches namespace on uninstall; it may contain other resources
		if obj.GetKind() == "Namespace" && obj.GetName() == DefaultNamespace {
			fmt.Printf("Skipping deletion of namespace %s (preserved for other resources)\n", DefaultNamespace)
			continue
		}

		// Delete the resource
		if err := i.client.Delete(ctx, obj); err != nil {
			return client.IgnoreNotFound(err)
		}
		fmt.Printf("Deleted resource %d: %s/%s\n", idx, obj.GetKind(), obj.GetName())
	}
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

		// Split multi-document YAML
		documents := strings.Split(resourceYAML, "\n---\n")

		for _, doc := range documents {
			if strings.TrimSpace(doc) == "" {
				continue
			}
			// Parse YAML to get resource info
			obj := &unstructured.Unstructured{}
			decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

			if _, _, err := decoder.Decode([]byte(doc), nil, obj); err != nil {
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
