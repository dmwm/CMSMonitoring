#!/bin/sh
##H Script to create CMS Eos path sizes with conditons
##H CMSVOC and CMSMONITORING groups are responsible for this script.

. /cvmfs/sft.cern.ch/lcg/views/LCG_96python3/x86_64-centos7-gcc8-opt/setup.sh

# Check xrdcp executable exists
if ! [ -x "$(command -v xrdcp)" ]; then
  echo "It seems xrdcp is not exist in PATH! Exiting..."
  exit 1
fi

if ! [ "$(python -c 'import sys; print(sys.version_info.major)')" = 3 ]; then
  echo "It seem python version is not 3.X! Exiting..."
fi

xrdcp root://eoscms.cern.ch//eos/cms/proc/accounting - |
  python $HOME/CMSMonitoring/src/python/CMSMonitoring/eos_path_size.py \
    --output_folder=/eos/user/c/cmsmonit/www/eos-path-size/size.html
