// MongoDB index creation. Indexing is the power source of the performance of MongoDB. There can be only 1 text index for 1 collection.

// switch db, default: rucio. It is replaced with sed command.
use _MONGOWRITEDB_;

// datasets indexes
db.main_datasets.createIndex( { "_id": 1, "Dataset": 1 } );
db.main_datasets.createIndex( { "Dataset": 1 } );
db.main_datasets.createIndex( { "LastAccess": 1 } );
db.main_datasets.createIndex( { "Max": 1 } );
db.main_datasets.createIndex( { "Min": 1 } );
db.main_datasets.createIndex( { "Avg": 1 } );
db.main_datasets.createIndex( { "Sum": 1 } );
db.main_datasets.createIndex( { "RealSize": 1 } );
db.main_datasets.createIndex( { "TotalFileCnt": 1 } );
db.main_datasets.createIndex( { "RseType": "text", "Dataset": "text", "RSEs": "text"} );

// detailed_datasets indexes
db.detailed_datasets.createIndex( { "_id": 1, "Type": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "Dataset": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "RSE": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "LastAccess": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "Dataset": 1, "Type": 1, "RSE":1 } );
db.detailed_datasets.createIndex( { "_id": 1, "ProdAccounts": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "Dataset": 1, "Type": 1, "RSE":1, "ProdAccounts": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "BlockRuleIDs": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "Dataset": 1, "Type": 1, "RSE":1, "BlockRuleIDs": 1 } );
db.detailed_datasets.createIndex( { "_id": 1, "BlockRuleIDs": 1, "ProdAccounts": 1 } );
db.detailed_datasets.createIndex( {"Type": "text", "Dataset": "text", "RSE": "text", "Tier": "text", "C": "text", "RseKind": "text"} );


// datasets_in_tape_and_disk indexes
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1 } );
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1 } );
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "TapeRseSet": 1 } );
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "DiskRseSet": 1 } );
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1, "TapeRseSet": 1 } );
db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1, "DiskRseSet": 1 } );
// Cannot index parallel 2 array column
//db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1, "TapeRseSet": 1, "DiskRseSet": 1 } );
