# Chart Values ConfigMap Feature

## Overview

The Helm Operator automatically creates ConfigMaps containing the `values.yaml` content for each chart version when a HelmRepository is synchronized. This feature provides easy access to chart default values for reference and customization purposes.

## How It Works

When a HelmRepository is synced, the operator:

1. **Discovers Charts**: Gets all charts from the repository
2. **Fetches All Versions**: For each chart, retrieves all available versions
3. **Downloads Chart Values**: Downloads each chart version and extracts the `values.yaml`
4. **Creates ConfigMaps**: Creates a ConfigMap for each chart version containing the values
5. **Sets Owner References**: Links ConfigMaps to the HelmRepository for automatic cleanup

## ConfigMap Naming Convention

ConfigMaps are named using the pattern:

```
helm-values-{repository}-{chart}-{version}
```

Examples:

- `helm-values-bitnami-nginx-1-0-0`
- `helm-values-stable-postgresql-12-1-5`
- `helm-values-my-repo-my-chart-v2-0-0-beta-1`

## ConfigMap Structure

Each generated ConfigMap includes:

### Labels

- `app.kubernetes.io/name: helm-operator`
- `app.kubernetes.io/component: chart-values`
- `helm-operator.ketches.cn/repository: {repository-name}`
- `helm-operator.ketches.cn/chart: {chart-name}`
- `helm-operator.ketches.cn/version: {chart-version}`

### Annotations

- `helm-operator.ketches.cn/chart-name: {chart-name}`
- `helm-operator.ketches.cn/chart-version: {chart-version}`
- `helm-operator.ketches.cn/repository: {repository-name}`

### Data

- `values.yaml`: The complete values.yaml content from the chart

### Owner Reference

- Automatically set to the parent HelmRepository
- Ensures ConfigMaps are deleted when the HelmRepository is deleted

## Example Usage

### 1. Create a HelmRepository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: default
spec:
  url: https://charts.bitnami.com/bitnami
  interval: 1h
```

### 2. ConfigMaps Are Created Automatically

After synchronization, you'll see ConfigMaps like:

```bash
kubectl get configmaps -l helm-operator.ketches.cn/repository=bitnami
```

### 3. View Chart Values

```bash
kubectl get configmap helm-values-bitnami-nginx-1-0-0 -o yaml
```

### 4. Use Values in HelmRelease

You can reference these values when creating HelmReleases:

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: my-nginx
spec:
  chart:
    name: nginx
    version: "1.0.0"
    repository:
      name: bitnami
  # You can now easily reference the default values from the ConfigMap
  # and customize only what you need
  values: |
    replicaCount: 3
    service:
      type: LoadBalancer
    # ... other customizations
```

## Querying ConfigMaps

### Find All Chart Values ConfigMaps

```bash
kubectl get configmaps -l app.kubernetes.io/component=chart-values
```

### Find ConfigMaps for a Specific Repository

```bash
kubectl get configmaps -l helm-operator.ketches.cn/repository=bitnami
```

### Find ConfigMaps for a Specific Chart

```bash
kubectl get configmaps -l helm-operator.ketches.cn/chart=nginx
```

### Find ConfigMaps for a Specific Version

```bash
kubectl get configmaps -l helm-operator.ketches.cn/version=1-0-0
```

## Lifecycle Management

### Automatic Creation

- ConfigMaps are created during HelmRepository synchronization
- If a ConfigMap already exists, it's updated if the values have changed

### Automatic Updates

- When a repository is re-synced, ConfigMaps are updated with the latest values
- New chart versions result in new ConfigMaps
- Removed chart versions result in ConfigMap deletion (if no longer present in the repository)

### Automatic Cleanup

- When a HelmRepository is deleted, all associated ConfigMaps are automatically deleted
- This is achieved through Kubernetes OwnerReference mechanism

## Benefits

1. **Easy Access**: Chart default values are readily available as Kubernetes resources
2. **Version Management**: Each chart version has its own ConfigMap
3. **Integration**: Can be easily referenced by other Kubernetes resources
4. **Automatic Cleanup**: No manual cleanup required when repositories are removed
5. **Searchability**: Use labels and selectors to find specific chart values
6. **Backup/Restore**: ConfigMaps can be backed up with standard Kubernetes tools

## Performance Considerations

- ConfigMap creation happens during repository synchronization
- Large repositories with many charts/versions may take longer to sync
- ConfigMaps are only created/updated when values actually change
- The operator uses efficient caching to minimize redundant downloads

## Troubleshooting

### ConfigMaps Not Created

1. Check HelmRepository status for sync errors
2. Verify repository URL is accessible
3. Check operator logs for detailed error messages

### ConfigMaps Not Updated

1. Ensure repository sync interval has passed
2. Check if repository sync is suspended
3. Verify operator has necessary RBAC permissions for ConfigMaps

### Missing Values

1. Some charts may not have a `values.yaml` file
2. The operator will use chart metadata default values as fallback
3. Check the chart source to verify values.yaml exists

## RBAC Requirements

The operator requires the following additional permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: helm-operator-configmap-manager
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

This is automatically included in the operator's default RBAC configuration.
