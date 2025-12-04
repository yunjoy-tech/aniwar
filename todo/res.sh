#!/usr/bin/env bash

echo ">>>>>>>>> 更新musae"
cd ../musae/
git pull
cd -

echo ">>>>>>>>> 更新protocol"
cd src/proto/protocol/
git pull
cd -


echo ">>>>>>>>> begin excel"
DoUpdateExcel=$1
if [ ${DoUpdateExcel} ==  "0" ]
then
    echo "DO NOT update ../design/excelSource"
else
    echo "update ../design/excelSource"
    svn revert -R ../design/excelSource
    svn cleanup .
    svn up ../design/excelSource
fi

cd ./tools/auto-export-excel-tool
bash ./start.sh
cd -
echo ">>>>>>>>> end excel"

echo ">>>>>>>>> begin proto"
cd ./src/proto
bash ./build.sh
cd -
echo ">>>>>>>>> end proto"

echo ">>>>>>>>> begin battleserver"
cd ./src/battleserver/Protos
bash ./build.sh
cd -
echo ">>>>>>>>> end battleserver"

sleep 5
exit
