#！/usr/local/env bash

JS_SCRIPT=$1
USAGE="Usage: $0 [JS_SCRIPT]"

if [ -z "${JS_SCRIPT}" ]; then
    echo "JS_SCRIPT must be set"
    echo $USAGE
    exit 1
fi
../output/bin/linux/mongosh --host s-uf698250405f7d94-pub.mongodb.rds.aliyuncs.com --port 3717 -u root -p 0XjLawDmTfo5IVeP --authenticationDatabase admin < ${JS_SCRIPT}