@echo off

docker run -d ^
  --name redis-dev ^
  -p 6377:6377 ^
  -e TZ=Asia/Shanghai ^
  --restart unless-stopped ^
  redis:5.0-alpine ^
  redis-server ^
    --appendonly yes ^
    --requirepass "123456" ^
    --loglevel notice ^
    --bind 0.0.0.0 ^
    --protected-mode no

echo "Redis started ..."