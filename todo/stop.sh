#!/usr/bin/env bash

killall daprd
killall dapr
killall consul
killall placement
# killall promtail

# logmonitorPID=`ps aux |grep -i script/logmonitor.py |grep -v grep |awk '{print $2}'`
# if [ $logmonitorPID -ne 0 ]; then kill -9 $logmonitorPID; fi
