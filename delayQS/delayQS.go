package delayQS

import (
	"fmt"
	"github.com/cihub/seelog"
	"github.com/spf13/viper"
	"github.com/youtube/vitess/go/pools"
	"golang.org/x/net/context"
	"io/ioutil"
	"os"
	"sync"
	"time"
)


var (
	logger      seelog.LoggerInterface
	pool        *pools.ResourcePool
	ctx         context.Context
	initMutex   sync.Mutex
	initialized bool
	stop string
	restart string
	retryNum map[string]int
	maxRetryNum int
	jobs chan delayJob
	tryJobs chan delayJob
)

func Init() error {
	initMutex.Lock()
	defer initMutex.Unlock()
	if !initialized {
		pid := os.Getpid()
		err := WritePid( viper.GetString("pid"), pid )
		if err != nil {
			fmt.Println("Error:", err )
			return  err
		}

		var logerr error
		logger,logerr = seelog.LoggerFromConfigAsFile("conf/config.xml")
		if logerr != nil{
			return logerr
		}

		ctx = context.Background()
		pool = newRedisPool(viper.GetString("redis"), viper.GetInt("connections"), viper.GetInt("connections"), time.Minute )

		retryNum = map[string]int{}
		maxRetryNum = viper.GetInt("maxRetryNum")
		initialized = true
	}

	return nil
}

func GetConn() (*RedisConn, error) {
	resource, err := pool.Get(ctx)

	if err != nil {
		return nil, err
	}
	return resource.(*RedisConn), nil
}

func PutConn(conn *RedisConn) {
	pool.Put(conn)
}

func Close() {
	initMutex.Lock()
	defer initMutex.Unlock()
	if initialized {
		pool.Close()
		initialized = false
	}
}

//写文件
func WritePid(name string, pid int) error {
	return  ioutil.WriteFile(name, []byte(fmt.Sprintln(pid)),0666)
}

func Run() error {
	err := Init()
	if err != nil{
		return  err
	}
	defer logger.Flush()

	logger.Info("delayQS started \n=================================================================================================================================================")

	quits := signals()
	key := viper.GetString("key")
	startTime := viper.GetInt("startTime")
	concurrency := viper.GetInt("concurrency") //并发执行数量
	retryIdKey := viper.GetString( "retryIdKey" ) //需要重试的任务id队列
	bodyKey := viper.GetString( "bodyKey" )
	failKey := viper.GetString( "failKey" ) //执行失败的任务
	retryInterval := viper.GetInt( "retryInterval" ) //提取失败任务的时间间隔

	jobs = make(chan delayJob, 10 )
	tryJobs = make( chan delayJob, 5 )

	var monitor sync.WaitGroup
	if err := getJob( &key, &startTime, quits[0], &monitor, &bodyKey ); err != nil{
		return  err
	}

	//失败的任务 重新尝试请求
	if err := reTry( &retryIdKey, &bodyKey, &retryInterval, &monitor, quits[1] ); err != nil{
		return  err
	}
	doJob( tryJobs, &monitor, &key, &retryIdKey, &bodyKey,  &failKey )

	//协程并发执行
	for i := 0; i<concurrency; i++ {
		doJob( jobs, &monitor, &key, &retryIdKey, &bodyKey,  &failKey )
	}



	monitor.Wait()
	logger.Info("delayQS stoped \n=================================================================================================================================================")
	return nil
}