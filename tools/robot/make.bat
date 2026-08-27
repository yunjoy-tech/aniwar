@echo off

echo start build robot ...
go build -gcflags "-N -l" -o ./robot.exe
