#!/usr/bin/env bash

read -p "Enter mongodb address: " mongodb_addr
read -s -p "Enter mongodb password: " mongodb_password

dbs=( account game mail gmt )
for db in ${dbs[@]}
do
    mongodump -h ${mongodb_addr} -u root -p ${mongodb_password} --authenticationDatabase admin -d ${db} -o mongo-backup
    echo "export db $s to ./mongo-backup/${db}"
done
