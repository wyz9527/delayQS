package delayQS

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"sync"
	"time"
)

//失败的重试
func reTry( retryIdKey *string, bodyKey *string,  retryInterval *int, monitor *sync.WaitGroup, quit <-chan bool) error{
	monitor.Add(1)
	go func() {
		defer func() {
			monitor.Done()
			close(tryJobs)
			if err := recover(); err !=nil{
				logger.Error( err )
			}

		}()

		for {

			select {
				case <-quit:
					logger.Info("接收到退出信号，获取重试任务动作退出")
					return
				default:
					conn,err := GetConn()
					if err != nil {
						panic(err)
						return
					}

					retryId, _:= redis.String( conn.Do( "LPOP", *retryIdKey ) )
					if retryId != "" {
						jobstr, err := redis.String( conn.Do("HGET", *bodyKey, retryId ) )
						if err != nil{
							PutConn( conn )
							panic(err)
							return
						}
						PutConn( conn )
						job := delayJob{}
						json.Unmarshal( []byte(jobstr), &job )
						job.Id = retryId
						tryJobs<-job
						logger.Info("重试任务加入tryJobs通道, 任务信息：", job)
					}else{
						PutConn( conn )
						timeout := time.After(time.Second * time.Duration( *retryInterval ))
						select {
							case <-quit:
								logger.Info("接收到退出信号，获取重试任务动作退出")
								return
							case <-timeout:
						}
					}


			}
		}
	}()
	return nil
}