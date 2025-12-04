start "filebeat-metric" /min ./output/bin/win/filebeat-metric.exe -e -c ./output/cfg/filebeat-metric.yml --path.data  ./filebeat-metric --path.home ./ --path.logs ./log/mlog/*
