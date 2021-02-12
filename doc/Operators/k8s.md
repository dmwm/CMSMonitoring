# The k8s clusters

The CMS Monitoring cluster is deployed on CERN OpenStack Kubernetes infrastructure.

## Setup a cluster

The full instructions of how to setup the cluster can be found in 
[CMSKubernetes/kubernetes/monitoring](https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/monitoring)
repository.

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
4. perform various k8s commands, e.g. check node/pods health
```
# obtain cluster config file
export KUBECONFIG=/<path>/config
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

For more information about kubernetes commands please consult CMSKubernetes
[tutorials](https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/cmsweb/docs)
