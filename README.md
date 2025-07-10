## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

  helm repo add helm-operator <https://ketches.github.io/helm-operator>

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
helm-operator` to see the charts.

To install the nginx chart:

    helm install nginx helm-operator/nginx

To uninstall the chart:

    helm uninstall nginx
