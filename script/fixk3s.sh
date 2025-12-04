#!/usr/bin/env bash


# echo "restart k3s"
# systemctl restart k3s

# sleep 5
/usr/local/bin/kubectl get pod -A | awk 'BEGIN {podCount=0} {split($0,a," "); if (NR>1 && a[0]!="aniwar"  && (a[4] ~ /CrashLoopBackOff|CreateContainerError|Error|Completed/ ||  a[3] ~ /0/)) {print "/usr/local/bin/kubectl delete pod " a[2] " --force  -n " a[1]; podCount+=1}} END { if (podCount>0) {print "systemctl restart k3s"}}' | sh
echo fixk3s `date` >> /tmp/cron.log