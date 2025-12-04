#!/usr/bin/python3
# encoding=utf-8
import os
import subprocess
import time
import requests
import json
from datetime import datetime as dt
import argparse
import sys
import threading
import numpy as np

url = 'https://open.feishu.cn/open-apis/bot/v2/hook/97f1f386-45e8-429e-9f82-cfbade8a3d5e'
# 获取目录下的文件log
file = os.popen("ls -l /data/server/log/plog/*.log|awk -F' ' '{print $9}'").read()
# print("file:", file)
logFiles = file.split('\n')
# print("logFiles:", logFiles)

MoreLines=10

WARN = "WARN"
ERROR = "ERROR"
FATAL = "FATAL"
CallStack = "CallStack"

# 检测的关键字数组，大小写敏感
keywords = []
keywords.append(WARN)
keywords.append(ERROR)
keywords.append(FATAL)
keywords.append(CallStack)



# keywords.append("CallStack")


class ArrayQueue():
    def __init__(self, maxsize=15):
        super(ArrayQueue, self).__init__()
        self.list = []
        self.maxSize = maxsize
        self.front = 0  # 指向 队列头部 原来等于-1的时候表示等于第一个数据的前一个
        self.rear = 0  # 指向队列尾部的值的后一个位置

    # 判断队列中是否已满
    def isfull(self):
        # return self.rear == self.maxSize - 1
        p = (self.rear + 1) % self.maxSize == self.front
        return p

    # 判断队列中是否已空
    def isEmpty(self):
        return self.rear == self.front

    # 增加队列的数据
    def addQueue(self, x):
        if self.isfull():
            self.getQueue()
        self.list.append(x)
        self.rear = (self.rear + 1) % self.maxSize  # 因为有可能直接到前面的位置

    # 取出队列中的数据
    def getQueue(self):
        if self.isEmpty():
            # print("队列已经空了，不能取数据")
            return
        # 先把front对应的值保存到一个临时变量
        # front后移
        # 将临时变量返回
        data = self.list.pop()
        self.front = (self.front + 1) % self.maxSize
        # print("队列取出来的数据为：", data)
        return data

    # 显示队列中数据
    def showlist(self):
        if self.isEmpty():
            #print("队列已经空了，不能取数据")
            return
        for i in range(self.front, self.numqueue()):
            print("列表的中的数据为{}".format(i % self.maxSize), self.list[i % self.maxSize])

    # get all text
    def getAllText(self):
        if self.isEmpty():
            #print("队列已经空了，不能取数据")
            return
        #newList = list(reversed(self.list))
        #self.list.reverse()
        allText = ""
        szLine = self.getQueue()
        while szLine:
            allText = allText + "\n\n" + szLine
            szLine = self.getQueue()
        return allText

    # 返回当前循环队列中的有效数据的个数
    def numqueue(self):
        return (self.rear + self.maxSize - self.front) % self.maxSize

    def headQueue(self):
        if self.isEmpty():
            #print("队列已经空了，不能取数据")
            return
        print("头部数据为：", self.list[self.front])


def send_msg(content, level):
    headers = {'Content-Type': 'application/json;charset=utf-8'}
    color = "yellow"
    title = "⚠警告日志"
    if level == ERROR or level == FATAL or level == CallStack:
        color = "red"
        title = "❌错误日志"
    data = {
        "msg_type": "interactive",
        "card": {
            "config": {
                "wide_screen_mode": True
            },
            "elements": [
                {
                    "tag": "div",
                    "text": {
                        "content": "\n{0}".format(content),
                        "tag": "lark_md"
                    }
                }
            ],
            "header": {
                "template": color,
                "title": {
                    "content": title,
                    "tag": "plain_text"
                }
            }
        }
    }

    r = requests.post(url, data=json.dumps(data), headers=headers)
    return r.text


def sendMsg(logFile, queue, level):
    szStr = "=== {0} ===\n\n".format(str(level))
    szStr = szStr + "\n\n{0}".format(str(queue.getAllText()))
    szStr = szStr + "\n\nlogfile:" + str(logFile)
    szStr = szStr + "\n\nhttp://192.168.2.19:5601/goto/787272e0-b28e-11ed-8224-8f8b4e75758c"
    send_msg(szStr, level)

# 检查行
def checkLine(logFile, line, arrayqueue, callStackQueue, isCallStach):
    if CallStack in line:
        isCallStach[0] = True
    if isCallStach[0]:
        callStackQueue.addQueue(line)
    if callStackQueue.isfull():
        isCallStach[0] = False
        sendMsg(logFile, line, callStackQueue, CallStack)
    for keyword in keywords:
        if keyword in line:
            sendMsg(logFile, line, arrayqueue, keyword)
            break

# 检查行
def checkLineEx(logFile, line, arrayqueue):
    for keyword in keywords:
        if keyword in line:
            sendMsg(logFile, arrayqueue, keyword)
            break

# 日志文件一般是按天产生，则通过在程序中判断文件的产生日期与当前时间，更换监控的日志文件
# 程序只是简单的示例一下，监控test1.log 10秒，转向监控test2.log
def monitorLog(logFile, index):
    # print('监控的日志文件 是%s' % logFile)
    popen = subprocess.Popen('tail -1f ' + logFile, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)
    pid = popen.pid
    print('Popen.pid:' + str(pid))
    arrayqueue = ArrayQueue()
    #callStackQueue = ArrayQueue()
    while True:
        line = popen.stdout.readline().strip()
        # 判断内容是否为空
        if line:
            arrayqueue.addQueue(str(line.decode('utf-8')))
            checkLineEx(logFile, str(line.decode('utf-8')), arrayqueue)
            #checkLine(logFile, str(line.decode('utf-8')), arrayqueue, callStackQueue, isCallStach)



if __name__ == '__main__':
    for index, logFile in enumerate(logFiles):
        tt = threading.Thread(target=monitorLog, args=(logFile, index))
        tt.start()
        print("monitor:" + str(logFile))
