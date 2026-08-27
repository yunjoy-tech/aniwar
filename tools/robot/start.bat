@echo off

set AccountPostfix="%1"
start "robot" robot.exe run --conf conf/config.json --accountTag robot1_%AccountPostfix% --accountBase "test" --accountStartIdx 3 --clientNum 1
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot2_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot3_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot4_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot5_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot6_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot7_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot8_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot9_%AccountPostfix% --accountBase 1  --clientNum 100
::start "robot" output\bin\win\robot.exe run --conf output/res/robot.conf --accountTag robot10_%AccountPostfix% --accountBase 1  --clientNum 100