### Example DataTable Json request

Reference: [datatables.net/manual/server-side](https://datatables.net/manual/server-side)

```json
{
    "draw": 1,
    "columns": [
        {
            "data": "Id",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "RseType",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "Dataset",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "LastAccess",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "LastAccessMs",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "MaxDatasetSizeInRsesGB",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "MinDatasetSizeInRsesGB",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "SumDatasetSizeInRsesGB",
            "name": "",
            "searchable": true,
            "orderable": true,
            "search": {
                "value": "",
                "regex": false
            }
        },
        {
            "data": "RSEs",
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
    "length": 20,
    "search": {
        "value": "",
        "regex": false
    },
    "custom": {
        "dataset": "",
        "rse": "",
        "tier": "",
        "rseCountry": "",
        "rseKind": "",
        "accounts": [],
        "rseType": []
    }
}
```

### Example SearchBuilder request

```json
{
    "criteria":
    [
        {
            "condition": "contains",
            "data": "Rse Type",
            "origData": "RseType",
            "type": "string",
            "value":
            [
                "asdasdas"
            ]
        },
        {
            "condition": "contains",
            "data": "Dataset",
            "origData": "Dataset",
            "type": "string",
            "value":
            [
                "asdasd"
            ]
        },
        {
            "condition": "<",
            "data": "Last Access",
            "origData": "LastAccess",
            "type": "date",
            "value":
            [
                "2022-07-05"
            ]
        },
        {
            "condition": ">",
            "data": "Last Access",
            "origData": "LastAccess",
            "type": "date",
            "value":
            [
                "2022-07-20"
            ]
        },
        {
            "condition": "between",
            "data": "Last Access",
            "origData": "LastAccess",
            "type": "date",
            "value":
            [
                "2022-07-05",
                "2022-07-21"
            ]
        },
        {
            "condition": "null",
            "data": "Last Access",
            "origData": "LastAccess",
            "type": "date",
            "value":
            []
        },
        {
            "condition": "!null",
            "data": "Last Access",
            "origData": "LastAccess",
            "type": "date",
            "value":
            []
        },
        {
            "condition": "<",
            "data": "Max",
            "origData": "Max",
            "type": "num",
            "value":
            [
                "11111"
            ]
        },
        {
            "condition": ">",
            "data": "Max",
            "origData": "Max",
            "type": "num",
            "value":
            [
                "2222222"
            ]
        },
        {
            "condition": "contains",
            "data": "RSEs",
            "origData": "RSEs",
            "type": "string",
            "value":
            [
                "sdassasada"
            ]
        }
    ],
    "logic": "AND"
}
```
