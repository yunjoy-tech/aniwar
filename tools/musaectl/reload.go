package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"os/exec"
	"strings"
)

type ReloadParam struct {
	// reload file type
	Type string `json:"type"` // conf,excel
	// reload files
	Files string `json:"files"` // 多个文件通过|符号分割
}

var Service string
var ReloadType string
var ReloadFiles string
var Timeout int

func init_reload() {
	reload := kingpin.Command("reload", "reload excel files")
	reload.Flag("svc", "service").Default("").StringVar(&Service)
	reload.Flag("type", "reload file type").Default("excel").StringVar(&ReloadType)
	reload.Flag("file", "reload files").Default("").StringVar(&ReloadFiles)
	reload.Flag("timeout", "timeout").Default("1").IntVar(&Timeout)
}

func cmd_reload() {
	sz := `kubectl -n aniwar get pod -o wide | awk '{split($0,a," "); if (NR>1) {printf "%s||%s\n",a[1],a[6]}}'`
	/*tOutput := `gmserver-844698995d-mtjgq||10.42.0.79
	guideserver-856cf97494-8rdbr||10.42.0.78
	battleserver-6c786f8479-p9jzj||10.42.0.81
	idipserver-7b949d9989-6lkml||10.42.0.80
	lobbyserver-f5bd49d48-mv4fz||10.42.0.82
	billserver-65b8866696-5w6sr||10.42.0.83
	loginserver-76498b8b59-csdmh||10.42.0.84
	gateserver-0||10.42.0.85
	actorserver-0||10.42.0.86`*/

	fmt.Println("svc:", Service)
	fmt.Println("type:", ReloadType)
	fmt.Println("file:", ReloadFiles)
	fmt.Println("timeout:", Timeout)
	reloadParam := &ReloadParam{
		Type:  ReloadType,
		Files: ReloadFiles,
	}
	reloadParamData, err := json.Marshal(reloadParam)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}
	cmd := exec.Command("bash", "-c", sz)
	if cmd != nil {
		fmt.Println(cmd.String())
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
			return
		} else {
			//fmt.Println(string(output))
			pods := strings.Split(string(output), "\n")
			for _, pod := range pods {
				svcIp := strings.Split(pod, "||")
				if len(svcIp) != 2 {
					fmt.Printf("pod:%s reolad failed\n", pod)
					continue
				}

				sub := strings.Split(svcIp[0], "-")
				if len(sub) < 2 {
					fmt.Printf("pod:%s reolad failed\n", pod)
					continue
				}
				svc := sub[0]
				if Service != "" && svc != Service {
					continue
				}
				var port int
				switch svc {
				case "actorserver":
					port = 24001
				case "idipserver":
					port = 29001
				default:
				}

				if port > 0 {
					url := fmt.Sprintf("http://%s:%d/api/hotReload", svcIp[1], port)
					ret, err := HttpPost(url, reloadParamData)
					if string(ret) != "SUCCESS" || err != nil {
						fmt.Printf("%s reolad failed, url:%s, data=%s\n", sub[0], url, string(reloadParamData))
					}
				}
			}
		}
	}
}
