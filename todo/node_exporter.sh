#!/usr/bin/env bash

./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/actor --web.listen-address=":9100"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/bill --web.listen-address=":9101"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/gate --web.listen-address=":9102"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/idip --web.listen-address=":9103"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/lobby --web.listen-address=":9104"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/login --web.listen-address=":9105"
# ./output/bin/linux/node_exporter  --collector.textfile.directory=log/mlog/mail --web.listen-address=":9106"
# ./output/bin/linux/node_exporter --config.file=cfg/node_exporter/exportercfg.yml