"""
Test script to translate condor_history command to Python using htcondor bindings.

Original command:
condor_history -name crab3@vocms0197.cern.ch -constraint '(JobUniverse == 5) && (CMS_Type != "DONOTMONIT") && (EnteredCurrentStatus >= '$(($(date +%s) - 3600))' || CRAB_PostJobLastUpdate >= '$(($(date +%s) - 3600))')' -pool cmsgwms-collector-global.fnal.gov:9620 > output.txt 2>&1 &
"""

import os
import time
import classad
import htcondor

# Configuration from the original command
SCHEDD_NAME = "crab3@vocms0197.cern.ch"
POOL = "cmsgwms-collector-global.fnal.gov:9620"

# Set collector host (equivalent to -pool option in condor_history)
# Must set BEFORE importing or creating Collector objects
os.environ["COLLECTOR_HOST"] = POOL

# Calculate time threshold (1 hour ago)
current_time = int(time.time())
time_threshold = current_time - 3600  # 3600 seconds = 1 hour

# Query collector for the specific schedd
# Since bash works, try using subprocess as a workaround if Python bindings fail
print(f"Querying collector {POOL} for schedd: {SCHEDD_NAME}")

schedd_ad = None
try:
    # Try with explicit collector first
    collector = htcondor.Collector(POOL)
    schedd_query = classad.ExprTree(f'Name == "{SCHEDD_NAME}"')
    schedds = collector.query(htcondor.AdTypes.Schedd, schedd_query)
    if schedds:
        schedd_ad = schedds[0]
        print(f"Found schedd via explicit collector connection")
except Exception as e:
    print(f"Explicit collector connection failed: {type(e).__name__}: {e}")
    print(f"Note: This is a known issue - Python bindings may have network/security restrictions")
    print(f"that don't affect the command-line tools.")
    print(f"\nSince bash commands work, you may need to:")
    print(f"  1. Check firewall rules for Python processes")
    print(f"  2. Check if Python bindings need additional security configuration")
    print(f"  3. Use subprocess to call condor_status as a workaround")
    exit(1)

if not schedd_ad:
    print(f"Error: Schedd {SCHEDD_NAME} not found in pool {POOL}")
    exit(1)

# Create Schedd object
schedd = htcondor.Schedd(schedd_ad)

# Build the constraint query
# (JobUniverse == 5) && (CMS_Type != "DONOTMONIT") && 
# (EnteredCurrentStatus >= time_threshold || CRAB_PostJobLastUpdate >= time_threshold)
constraint = f"""
    (JobUniverse == 5) && (CMS_Type != "DONOTMONIT")
    &&
    (
        EnteredCurrentStatus >= {time_threshold}
        || CRAB_PostJobLastUpdate >= {time_threshold}
    )
"""
history_query = classad.ExprTree(constraint)

print(f"Querying schedd: {SCHEDD_NAME}")
print(f"Pool: {POOL}")
print(f"Time threshold: {time_threshold} ({time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(time_threshold))})")
print(f"Constraint: {history_query}")
print("-" * 80)

try:
    # Time the history query itself
    query_start_time = time.time()
    history_iter = schedd.history(history_query, [], match=-1)
    query_end_time = time.time()
    query_duration = query_end_time - query_start_time
    
    print(f"History query initiated in {query_duration:.2f} seconds")

    
    # Output timing information
    print(f"\nTotal jobs found: {len(history_iter)}")
    print("\nTiming information:")
    print(f"  History query initiation: {query_duration:.2f} seconds")
    if len(history_iter) > 0:
        print(f"  Average time per job: {query_duration/len(history_iter):.4f} seconds")
    
except RuntimeError as e:
    print(f"Error querying history: {e}")
    exit(1)
