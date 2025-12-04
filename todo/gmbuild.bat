
echo "build server..."
pushd .
cd tools\gin-vue-admin\server
start  build.bat
popd

echo "build web..."
pushd .
cd tools\gin-vue-admin\web
start /wait build.bat
xcopy  dist ..\..\..\output\gm\dist /E /Y /S /I /F
popd


