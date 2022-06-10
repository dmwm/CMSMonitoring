## rucio-dataset-mon-go

### Requirement of the service

`GO_ENVS_SECRET_PATH` OS environment variable should be set which is the full path of ".env" file:

```
MONGOURI=mongodb://admin:password@cuzunogl-jhpbqba52z6h-node-0:32000/admin?retryWrites=true&w=majority
MONGO_DATABASE=rucio
MONGO_CONNECT_TIMEOUT=10
COLLECTION_DATASETS=datasets
```

### Example DataTable Json request

Reference: [datatables.net/manual/server-side](https://datatables.net/manual/server-side)

```json
{
    "draw": 2,
    "columns": [
        {
            "data": "dataset",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "rse",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "size",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        }
    ],
    "order": [
        {
            "column": 0,
            "dir": "asc"
        }
    ],
    "start": 0,
    "length": 5,
    "search": {
        "value": "s",
        "regex": false
    }
}
```

### References

- Reference: https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gin-gonic-version-269m
