
start "filebeat" ./output/bin/win/filebeat.exe -e -c ./output/cfg/filebeat.yml --path.data  ./filebeat --path.home ./ --path.logs ./log