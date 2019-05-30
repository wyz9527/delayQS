package delayQS

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/xid"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)


//获取到期的job
func getJob(jobKey *string, startTime *int, quit <-chan bool, monitor *sync.WaitGroup, bodyKey *string )   error  {

	monitor.Add(1 )
	go func() {
		defer func() {
			monitor.Done()
			close(jobs)
		}()
		for  {
			select {
			case <-quit:
				logger.Info("接收到退出信号，获取任务动作退出")
				return
			default:
				conn,err := GetConn()
				if err != nil {
					logger.Error( err )
					return
				}
				nowTime := time.Now().Unix()
				replay, err := redis.Strings( conn.Do("ZRANGEBYSCORE", *jobKey, *startTime, nowTime ) )
				if err != nil {
					PutConn(conn)
					logger.Error(err)
					return
				}
				for _, rp := range replay{
					//将任务加入通道
					job := delayJob{}
					json.Unmarshal( []byte(rp), &job )
					job.Id = xid.New().String()
					jobs<-job
					//加入到bodykey里面
					conn.Send("HSET", *bodyKey, job.Id, rp )
					logger.Info("\n成功获取一个job->", rp )
				}
				conn.Flush()
				PutConn(conn)

				*startTime = int( nowTime )+1
				if  time.Now().Unix() == nowTime{
					t := time.After(time.Second)
					<-t
				}
			}
		}
	}()

	return nil
}

//执行具体任务
func doJob( workJobs chan delayJob, monitor *sync.WaitGroup, jobKey *string, retryIdKey *string, bodyKey *string, failKey *string ) error {

	monitor.Add(1)
	go func() {
		defer func() {
			monitor.Done()
			if err := recover(); err != nil {
				logger.Error(err)
			}
		}()

		for job := range workJobs {
			conn, err := GetConn()
			if err != nil{
				panic(err)
				return
			}

			logger.Info("\n准备执行job->", job )

			//发送http请求
			byteBody, _ := json.Marshal( job.Body )
			resp, herr := httpPost( &job.Callback, string(byteBody), job.Sign )

			logger.Info("\n ================================================================================================================================================= \n " +
				"||发送POST请求：url: ", job.Callback, "||\n ||请求数据data:", string(byteBody), "||\n ||签名sign:", job.Sign, "|| \n " +
				"||请求结果：", resp, "||\n||错误信息: ", herr, "|| \n",
				"=================================================================================================================================================" )

			if herr != nil || strings.Index(strings.ToLower(resp), "success" ) < 0 {
				if retryNum[job.Id] > maxRetryNum{ //重试次数超过最大次
					logger.Info( job.Id, " -> 加入失败重试队列： ", *failKey )
					delete(retryNum,job.Id) //重试次数超过最大次,删除key
					//将数据加入到记录失败任务的key
					bytejob,_ := json.Marshal( job )
					conn.Send("RPUSH", *failKey, string( bytejob ) )
					//删除原生jobs集合里面的数据
					strjob, _ := redis.String( conn.Do("HGET", *bodyKey, job.Id ) )
					if strjob != ""{
						conn.Send("ZREM", *jobKey,  strjob )
					}
					//删除body key里面的值
					conn.Send("HDEL", *bodyKey, job.Id )

				}else{
					//任务id加入到重试队列
					logger.Info(job.Id , "->加入到重试队列:", *retryIdKey )
					conn.Send("RPUSH", *retryIdKey, job.Id )
					retryNum[job.Id]++
				}
			}else {
				//任务成功发送，且消费者返回成功信息
				strjob, _ := redis.String( conn.Do("HGET", *bodyKey, job.Id ) )
				if strjob != ""{
					conn.Send("ZREM", *jobKey,  strjob )
				}
				conn.Send("HDEL", *bodyKey, job.Id )
			}


			conn.Flush()
			PutConn(conn)
		}
	}()

	return  nil
}

func httpPost( posturl *string, data string, sign string ) ( string, error ) {
	postValues := url.Values{
		"body":{data},
		"sign":{sign},
	}
	resp, err := http.PostForm(*posturl, postValues) // Content-Type post请求必须设置
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), nil
}
