package delayQS

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var godaemon 	= flag.Bool("d", false, "run app as a daemon with -d=true")
var opt 		= flag.String("opt", "start", "-opt=start/stop/restart")

func init() {
	if !flag.Parsed() {
		flag.Parse()
	}
	if err := InitConf(""); err != nil {
		fmt.Println(err)
		os.Exit( 0 )
	}

	args := os.Args[1:]
	if *godaemon {
		i := 0
		for ; i < len(args); i++ {
			if args[i] == "-d=true" {
				args[i] = "-d=false"
				break
			}
		}
		if *opt == "start"{
			pidFile := viper.GetString("pid")
			pf, _ := os.OpenFile(pidFile, os.O_RDWR, 0)
			defer pf.Close()
			if pf != nil{
				pd, _ := ioutil.ReadAll( pf )
				old_pid, _ := strconv.Atoi( strings.Replace(string(pd),"\n","", -1) )
				if old_pid > 0{
					if err := procMethod( old_pid ); err == nil{
						fmt.Println("程序运行中，请不要重复启动")
						os.Exit(0)
					}
				}
			}

			cmd := exec.Command(os.Args[0], args...)
			cmd.Start()
			fmt.Println("[PID]", cmd.Process.Pid)
			os.Exit(0)
		}
	}

	if *opt != "start"{
		//读取pid
		pidFile := viper.GetString("pid")
		pf, err := os.OpenFile(pidFile, os.O_RDWR, 0)
		defer pf.Close()
		if os.IsNotExist(err){
			fmt.Println(pidFile, "文件不存在")
			os.Exit( 0 )
		} else if err != nil{
			fmt.Println(err)
			os.Exit(0)
		}
		pd, _ := ioutil.ReadAll( pf )

		old_pid, err := strconv.Atoi( strings.Replace(string(pd),"\n","", -1) )
		if err != nil{
			fmt.Println( err )
			os.Exit( 0 )
		}

		if *opt == "stop"{
			err := syscall.Kill(old_pid, syscall.SIGQUIT)
			if err == nil{
				fmt.Println("stop success")
			}
			os.Exit( 0 )

		}else if *opt == "restart"{
			err := syscall.Kill(old_pid, syscall.SIGQUIT)
			if err == nil{
				fmt.Println("stop success")
			}
			fmt.Println("starting...")
			after := time.After(time.Second*5)
			<-after

			cmd := exec.Command(os.Args[0], "-opt=start", "-d=false")
			cmd.Start()
			fmt.Println("[PID]", cmd.Process.Pid)
			fmt.Println("start success")
			os.Exit(0)
		}
	}

}

func procMethod(pid int) error {
	_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err
}