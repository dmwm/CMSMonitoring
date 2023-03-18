### Required environment variables

- `GO_ENVS_SECRET_PATH` : file path:

Should include below variables. `MONGOURI` credentials should be provided as below. All variables are required.

```
MONGOURI=mongodb://[USERNAME]:[PASSWORD]@mongodb-0.mongodb.cmsmon-mongo.svc.cluster.local:27017/[DATABASE]?retryWrites=true&w=majority
MONGO_ADMIN_DB=admin
MONGO_DATABASE=rucio
MONGO_CONNECT_TIMEOUT=10
COLLECTION_DATASETS=datasets
```

- `MONGO_ROOT_USERNAME` : MongoDB user which has r/w access
- `MONGO_ROOT_PASSWORD` : MongoDB user password which has r/w access

#### Kubernetes deployment

- https://github.com/dmwm/CMSKubernetes/tree/master/docker/rucio-dataset-monitoring
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/services/datasetmon
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/deploy-secrets.sh (see how required K8s secret
  is created)
