#!/usr/bin/env bash
chmod +x ./output/bin/linux/versionTool
# cd ./output/

if ${PUBLIC}; then
  pkgName=aniwar-server-${BRANCH}-${VERSION}-${DOCKER_TAG}-public-$(date "+%Y_%m_%d_%H_%M_%S")
else
  pkgName=aniwar-server-${BRANCH}-${VERSION}-${DOCKER_TAG}-local-$(date "+%Y_%m_%d_%H_%M_%S")
fi

fileName=${pkgName}.tar.gz
installName=install-${pkgName}.sh
uploadName=upload-${pkgName}.sh
versionDir=./pkg/${pkgName}

mkdir -m 755 -p ${versionDir}

echo "
#!/usr/bin/env bash
scp ./${fileName} tsh-aniwar-dev-jump-bastion:/home/aniwar/pkg/${fileName}
scp ./${installName} tsh-aniwar-dev-jump-bastion:/home/aniwar/pkg/${installName}
" > ./pkg/${pkgName}/${uploadName}

echo "
#!/usr/bin/env bash
sudo tar -zxvf ${fileName} -C /mnt/nas/k3s/govtest/app/server/
" > ./pkg/${pkgName}/${installName}

./output/bin/linux/versionTool build --version ${VERSION} --mode server \
  --file ./versionfiles.txt --output ${versionDir}/${fileName} \
  --ignore ./versionignore.txt

# if [ "${PUBLIC}" = "true" ]; then
if ${PUBLIC}; then
  echo "发布版本: "${fileName}
  /usr/bin/cp -rf ${versionDir} /data/nfs/version/server/
  /usr/bin/cp -rf ${versionDir} /data/server-version/
  cd /data/server-version/
  echo `pwd`
  git restore .
  git add .
  git commit -m "upload version: "${fileName}
  git push
  cd -
else
  echo "不发布版本: "${fileName}
  mv ${versionDir} /data/nfs/version/server/
fi

# 更新version list
echo ${pkgName} >> /data/nfs/version/server/list.txt

# cd -


