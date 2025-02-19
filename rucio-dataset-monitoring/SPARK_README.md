
### Example main_datasets result in HDFS

```json
{
  "Id": 14146298,
  "RseType": "DISK",
  "Dataset": "/Pseudoscalar_MonoV_...stic_v15-v2/MINIAODSIM",
  "LastAccess": 2147483647,
  "Max": "32980965507",
  "Min": "32980965507",
  "Avg": "32980965507",
  "Sum": "32980965507",
  "RealSize": 32980965507,
  "TotalFileCnt": 21,
  "RSEs": "T2_US_MIT"
}
```


### Example detailed_datasets (Datasets in each RSE) result JSON in HDFS

```json
{
  "Type": "DISK",
  "Dataset": "/ZprimeToTprim.....TrancheIV_v6-v2/MINIAODSIM",
  "RSE": "T2_US_MIT",
  "Tier": "T2",
  "C": "US",
  "RseKind": "prod",
  "SizeBytes": 15013007503,
  "LastAccess": 0,
  "IsFullyReplicated": true,
  "IsLocked": "FULLY",
  "FilePercentage": 100.00,
  "FileCount": 7,
  "AccessedFileCount": 0,
  "BlockCount": 2,
  "ProdLockedBlockCount": 2,
  "ProdAccounts": ["transfer_ops"],
  "BlockRuleIDs": ["7a2d9ace81ac490e852747cdeb540e36"]
}
```

### Example datasets_in_tape_disk result in HDFS

```json
{
    "Dataset": "/ADDGravToGG_MS-4500_....X_mcRun2_asymptotic_v3-v2/NANOAODSIM",
    "MaxSize": 157169408,
    "TapeFullyReplicatedRseCount": 1,
    "DiskFullyReplicatedRseCount": 2,
    "TapeFullyLockedRseCount": 1,
    "DiskFullyLockedRseCount": 2,
    "TapeRseCount": 1,
    "DiskRseCount": 2,
    "TapeRseSet": ["T0_CH_CERN_Tape"],
    "DiskRseSet": ["T2_FR_GRIF_IRFU", "T2_US_Nebraska"]
}
```
