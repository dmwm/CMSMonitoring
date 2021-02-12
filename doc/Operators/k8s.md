# The k8s clusters

The CMS Monitoring cluster is deployed on CERN OpenStack Kubernetes infrastructure.
If you want to understand basic concepts and see end-to-end deployment of a basic application to k8s cluster please follow this [tutorial](https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/cmsweb/docs/end-to-end.md).

## Setup the CMS monitoring cluster

The full instructions of how to setup the cluster can be found in 
[here](https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/monitoring/README.md).

## Cluster maintenance

The maintenance of the cluster is quite trivial. The operator expects to follow these steps:
1. login to `lxplus-cloud`
2. clone CMSKubernetes repository
```
git clone git@github.com:dmwm/CMSKubernetes.git
cd CMSKubernetes/kubernetes/monitoring
```
3. clone CMS Monitoring secrets repository
```
git clone https://:@gitlab.cern.ch:8443/cms-monitoring/secrets.git
```
4. Setup environment (cluster config file comes from secrets repository)
```
export OS_PROJECT_NAME="CMS Web"
export KUBECONFIG=/<path>/config
```
4. perform various k8s commands, e.g. check node/pods health
```
# check node status
kubectl get node
kubectl top node
# check pod status
kubectl get pods
kubectl top pods
# use deploy.sh script to get status of the cluster
./deploy.sh status
# explore logs of certain pods
kubectl logs <pod-name>
```

If a node gets stuck:
```
openstack server reboot --hard monitoring-cluster-ysvr3hqxecfu-master-0 
```

## CMSWeb k8s cluster

To learn about cmsweb k8s architecture please read [architecture](docs/architecture.md) document.

To deploy cmsweb on kubernetes please follow these steps:
- [cluster creation](docs/cluster.md)
- [general deployment](docs/deployment.md)
- [cmsweb deployment](docs/cmsweb-deployment.md)
- [nginx](docs/nginx.md)
- [autoscaling](docs/autoscaling.md)
- [storage](docs/storage.md) (this step is optional)
- [hadoop](docs/hadoop.md) (this step is optional)
- [troubleshooting](docs/troubleshooting.md)
- [references](docs/references.md)




If you want to understand basic concepts and see end-to-end deployment
of basic application to k8s cluster please follow this
[tutorial](docs/end-to-end.md)

