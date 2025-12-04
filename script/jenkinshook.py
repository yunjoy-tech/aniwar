import os
import requests
import json
from datetime import datetime as dt
import argparse
import sys

svnUsetArray = {
    "chenlianjia" : "7137874919165034498",
    "zhouchao" : "7137874897673633794",
    "yangyifan" : "7137874550433103874",
    "tangjingyi" : "7137876335531458561",
    "zhongxiyao" : "7137877762274131969",
    "zongnuo" : "7137877660061466652",
                }

def send_msg(_url, _msg,_state):
    """
    :param _url:
    :param _msg:
    :return:
    """
    headers = {'Content-Type': 'application/json;charset=utf-8'}
    if _state == 1 :
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
                        "content": "\n🕐时间：{0} {1}".format(dt.now().strftime('%Y-%m-%d %H:%M:%S'), _msg),
                        "tag": "lark_md"
                    }
                }
            ],

            "header": {
                "template": "red",
                "title": {
                    "content": "❌导表失败",
                    "tag": "plain_text"
                }
            }
           }
        }
    else:
        data = {
            # "msg_type": "text",
            # "content": {
            #     "text": "@所有人 \nJenkins自动导表\n\n时间：{0} {1}".format(dt.now().strftime('%Y-%m-%d %H:%M:%S'), _msg)
            # }
            "msg_type": "interactive",
            "card": {
                "config": {
                    "wide_screen_mode": True
                },
                "elements": [
                    {
                        "tag": "div",
                        "text": {
                            "content": "\n🕐时间：{0} {1}".format(dt.now().strftime('%Y-%m-%d %H:%M:%S'),_msg),
                            "tag": "lark_md"
                        }
                    }
                ],
                "header": {
                    "template": "green",
                    "title": {
                        "content": "✅导表成功",
                        "tag": "plain_text"
                    }
                }
            }
        }

    r = requests.post(_url, data=json.dumps(data), headers=headers)
    return r.text

def GetUserID(log):
    for userName in svnUsetArray:
        if userName in log:
            return svnUsetArray[userName]
    return ""

if __name__ == '__main__':
    # 注释掉的是不必要的调试用的
    # args_list = sys.argv
    # if args_list.__len__() < 2:
    #     print("加参数 --msg 后面跟要发送的消息")
    #     exit(1)
    #
    # arg1 = sys.argv[1]
    # if arg1 == '-h' or arg1 == '--help':
    #     print("加参数 --msg 后面跟要发送的消息")
    #     exit(0)

    svnLog = os.popen("svn log svn://192.168.2.15/design -v -l1").read()
    userID = GetUserID(svnLog)
    print(userID)
    parser = argparse.ArgumentParser(prog='feishu_alert',description='飞书消息通知消息')
    parser.add_argument('--state', type=int, default=None, required=True, help="状态")
    parser.add_argument('--num', type=str, default=None, required=True, help="状态")
    parser.add_argument('--console', type=str, default=None, required=True, help="状态")
    args = parser.parse_args()
    print(args)
    msg = ""
    if args.state == 1:
        msg = "\n📋日志：\n" + svnLog + "\n🔢编号：#"+args.num+"\n🗂️状态：失败"+ "\n👤提交人:<at id="+userID+"></at>" + "\n📍控制台：htt:" +  args.console
    if args.state == 0:
        msg = "\n📋日志：\n"+svnLog + "\n🔢编号：#"+args.num+"\n🗂️状态：成功\n" + "📍控制台：htt" +  args.console
    url = 'https://open.feishu.cn/open-apis/bot/v2/hook/a54b7f04-f51a-41a3-8549-e24d07c61430'
    print(send_msg(url, msg,args.state))

