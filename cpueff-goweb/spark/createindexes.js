// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// MongoDB index creation. Indexing is the power source of the performance of MongoDB. There can be only 1 text index for 1 collection.

// switch db, default: cpueff. It is replaced with sed command.
use _MONGOWRITEDB_;

// condor_main indexes
db.condor_main.createIndex( { "_id": 1, "Workflow": 1 } );
db.condor_main.createIndex( { "_id": 1, "WmagentRequestName": 1 } );
db.condor_main.createIndex( { "Workflow": 1 } );
db.condor_main.createIndex( { "WmagentRequestName": 1 } );
db.condor_main.createIndex( { "Workflow": "text", "WmagentRequestName": "text"} );

// condor_detailed indexes
db.condor_detailed.createIndex( { "_id": 1, "Workflow": 1 } );
db.condor_detailed.createIndex( { "_id": 1, "WmagentRequestName": 1 } );
db.condor_detailed.createIndex( { "Workflow": 1 } );
db.condor_detailed.createIndex( { "WmagentRequestName": 1 } );
db.condor_detailed.createIndex( { "Site": 1 } );
db.condor_detailed.createIndex( { "Workflow": "text", "WmagentRequestName": "text"} );

// sc_task indexes
db.sc_task.createIndex( { "_id": 1, "Task": 1 } );
db.sc_task.createIndex( { "Task": 1 } );

// sc_task_cmsrun_jobtype_site indexes
db.sc_task_cmsrun_jobtype_site.createIndex( { "_id": 1, "Task": 1 } );
db.sc_task_cmsrun_jobtype_site.createIndex( { "Task": 1 } );
db.sc_task_cmsrun_jobtype_site.createIndex( { "StepName": 1 } );
db.sc_task_cmsrun_jobtype_site.createIndex( { "JobType": 1 } );
db.sc_task_cmsrun_jobtype_site.createIndex( { "Site": 1 } );
db.sc_task_cmsrun_jobtype_site.createIndex( { "Task": "text", "StepName": "text", "JobType": "text", "Site": "text"} );

// Cannot index parallel 2 array column
//db.datasets_in_tape_and_disk.createIndex( { "_id": 1, "Dataset": 1, "TapeRseSet": 1, "DiskRseSet": 1 } );
