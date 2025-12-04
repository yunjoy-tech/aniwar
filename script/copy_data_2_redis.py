import sys
import redis
import os

fileName = sys.argv[1]
r = redis.StrictRedis(host='192.168.2.7')
with open(fileName) as f:
    value = f.read()
r.set('cfg:excelData:'+ fileName.split("/")[-1], value)
