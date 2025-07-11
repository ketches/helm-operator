# Helm Operator Installer Package

This package provides a programmatic way to install and manage the helm-operator in Kubernetes clusters.

## Features

- **Embedded Resources**: All required Kubernetes manifests are embedded in the binary
- **Programmatic Installation**: Install helm-operator using Go code
- **Status Checking**: Check installation status and resource health
- **Clean Uninstall**: Remove all helm-operator resources
- **Client Integration**: Works with any `controller-runtime` client

## Usage

### Basic Installation

```go
package main

import (
    "context"
    "log"
    
    "k8s.io/client-go/rest"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    "github.com/ketches/helm-operator/pkg/installer"
)

func main() {
    // Get Kubernetes client
    config, err := rest.InClusterConfig()
    if err != nil {
        log.Fatal(err)
    }
    
    k8sClient, err := client.New(config, client.Options{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create installer
    helmInstaller := installer.NewInstaller(k8sClient)
    
    // Install helm-operator
    ctx := context.Background()
    if err := helmInstaller.Install(ctx); err != nil {
        log.Fatalf("Installation failed: %v", err)
    }
    
    log.Println("helm-operator installed successfully!")
}
```

### Check Installation Status

```go
status, err := helmInstaller.GetStatus(ctx)
if err != nil {
    log.Fatal(err)
}

if status.Installed {
    log.Println("helm-operator is fully installed")
} else {
    log.Println("helm-operator is not fully installed")
}

// Check individual resources
for _, resource := range status.Resources {
    log.Printf("Resource %s/%s: exists=%v", 
        resource.Kind, resource.Name, resource.Exists)
}
```

### Uninstall

```go
if err := helmInstaller.Uninstall(ctx); err != nil {
    log.Fatalf("Uninstallation failed: %v", err)
}

log.Println("helm-operator uninstalled successfully!")
```

## Embedded Resources

The installer includes the following Kubernetes resources:

1. **Custom Resource Definitions (CRDs)**
   - `HelmRepository` CRD
   - `HelmRelease` CRD

2. **RBAC Resources**
   - ServiceAccount
   - ClusterRole
   - ClusterRoleBinding

3. **Deployment**
   - helm-operator controller deployment
   - Required environment variables and configurations

4. **Additional Resources**
   - ConfigMaps (if any)
   - Services (if any)

## Resource Generation

The embedded resources are automatically generated from the `deploy/` directory during the release process:

```bash
# Manually regenerate resources
make generate-resources

# Resources are automatically regenerated during release
make release-complete VERSION=0.3.0
```

## API Reference

### `HelmOperatorInstaller`

The main installer struct that provides installation methods.

#### Methods

- `NewInstaller(client client.Client) *HelmOperatorInstaller`
  - Creates a new installer instance

- `Install(ctx context.Context) error`
  - Installs all helm-operator resources

- `Uninstall(ctx context.Context) error`
  - Removes all helm-operator resources

- `GetStatus(ctx context.Context) (*InstallationStatus, error)`
  - Returns the current installation status

### `InstallationStatus`

Represents the status of the helm-operator installation.

```go
type InstallationStatus struct {
    Installed bool             `json:"installed"`
    Resources []ResourceStatus `json:"resources"`
}
```

### `ResourceStatus`

Represents the status of an individual Kubernetes resource.

```go
type ResourceStatus struct {
    Index     int    `json:"index"`
    Kind      string `json:"kind"`
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
    Exists    bool   `json:"exists"`
    Error     string `json:"error,omitempty"`
}
```

## Error Handling

The installer provides detailed error information:

```go
if err := helmInstaller.Install(ctx); err != nil {
    log.Printf("Installation failed: %v", err)
    
    // Check status to see which resources failed
    status, _ := helmInstaller.GetStatus(ctx)
    for _, resource := range status.Resources {
        if resource.Error != "" {
            log.Printf("Resource %s/%s failed: %s", 
                resource.Kind, resource.Name, resource.Error)
        }
    }
}
```

## Integration Examples

### With Operator SDK

```go
func (r *MyReconciler) ensureHelmOperator(ctx context.Context) error {
    installer := installer.NewInstaller(r.Client)
    
    status, err := installer.GetStatus(ctx)
    if err != nil {
        return err
    }
    
    if !status.Installed {
        return installer.Install(ctx)
    }
    
    return nil
}
```

### With Custom Controllers

```go
type HelmOperatorManager struct {
    client    client.Client
    installer *installer.HelmOperatorInstaller
}

func NewHelmOperatorManager(client client.Client) *HelmOperatorManager {
    return &HelmOperatorManager{
        client:    client,
        installer: installer.NewInstaller(client),
    }
}

func (m *HelmOperatorManager) EnsureInstalled(ctx context.Context) error {
    return m.installer.Install(ctx)
}
```

## Best Practices

1. **Check Status First**: Always check installation status before installing
2. **Handle Errors**: Properly handle and log installation errors
3. **Use Context**: Always pass context for cancellation support
4. **Resource Cleanup**: Use the uninstall method for clean removal
5. **Version Compatibility**: Ensure the installer version matches your requirements

## Troubleshooting

### Common Issues

1. **Permission Denied**
   - Ensure the client has sufficient RBAC permissions
   - Check if the ServiceAccount has cluster-admin or required permissions

2. **Resource Already Exists**
   - The installer handles existing resources gracefully
   - Use `GetStatus()` to check current state

3. **Network Issues**
   - Ensure connectivity to the Kubernetes API server
   - Check if the cluster is accessible

### Debug Mode

Enable debug logging to see detailed installation progress:

```go
// The installer automatically logs progress to stdout
// Check the logs for detailed information about each resource
```

## Contributing

When adding new resources to the `deploy/` directory:

1. Add your YAML files to the appropriate subdirectory
2. Run `make generate-resources` to update the embedded resources
3. Test the installation with the new resources
4. Update this documentation if needed

## License

This package is part of the helm-operator project and follows the same license terms.
