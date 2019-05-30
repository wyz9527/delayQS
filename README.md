# delayQS
go+redis 做的一个延迟任务处理

## 适用业务场景
需要在某一个时间点去处理某项业务。比如一个未付款订单的有效期是3天，3天后要将订单状态改成关闭。

## 配置
参考config文件夹下面的配置文件
config.json -- 基础配置
config.xml  -- seelog配置

### config.json 配置说明
```javascript
{
  "redis": "redis://localhost:6379/", //redis链接dns
  "connections": 4, //redis链接数
  "concurrency": 2, //并发执行的携程数
  "startTime": 0, //第一次获取任务信息的时的score值
  "key": "DelayQs:job_zqueue", //redis key  有序集合 格式是 value=job[要执行的任务] score=timestamp[任务的执行时间点，秒级时间戳]
  "retryIdKey": "DelayQs:retry_id", //redis key 队列 需要重试的任务id
  "bodyKey": "DelayQs:body", //redis key hash key  任务id=> 任务信息
  "failKey": "DelayQs:fail_job", //redis key  队列 尝试多次后执行失败的任务信息
  "maxRetryNum": 3, //任务重新执行，最大尝试次数
  "retryInterval" : 5, //获取重试任务的时间间隔
  "pid" : "/var/log/delayQs/delayqs.pid" //pid 保存文件
}
```

### config.xml 配置说明
```xml
<seelog type="asynctimer" asyncinterval="5000000" minlevel="trace" maxlevel="error">
    <outputs formatid="info">
        <filter levels="error"> <!--错误信息的配置-->
            <buffered formatid="info" size="10000" flushperiod="1000">
                <rollingfile type="date" filename="./logs/error.log" datepattern="02.01.2006" fullname="true" maxrolls="30"/>
            </buffered>
        </filter>
        <filter levels="info"> <!--info信息的配置-->
            <buffered formatid="info" size="10000" flushperiod="1000">
                <rollingfile type="date" filename="./logs/info.log" datepattern="02.01.2006" fullname="true" maxrolls="30"/>
            </buffered>
        </filter>
    </outputs>

    <formats>
        <format id="info" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />
    </formats>
</seelog>
```
### 任务格式规定
···go
type delayJob struct {
	Id string //任务id
	Body interface{} `json:"body"` //任务的数据 json格式
	Callback string `json:"callback"` //回调地址 url
	Sign string `json:"sign"` //验证字符串
}
```

## 安装
```shell
go get -u github.com/wyz9527/delayQS/tree/master/delayQS
```

## 示列
模拟场景：将一个未付款订单信息，1小时后状态改成关闭
### 下单成功 corder.php 
```php
<?php
	$key = 'DelayQs:job_zqueue';
	$redis = new Redis();
	$redis->connect('127.0.0.1', 6379, 60));
	$orderId = 1; //订单id 
	$expireTime = time() + 3600; //过期时间
	//将需要处理的订单id加入到需要监听的redis 队列
	$data = [
		"body" => ["orderId"=>1],
		"callback" => "http://localhost/orderTimeOut.php" //任务执行地址
	];
	$privateKey = "sadhfusa7uy9iMn"; //签名加密密钥
	ksort($data["body"]);
	$sign = md5( $privateKey.implode('&',$data['body']) ); //生成签名
	$data['sign'] = $sign;
	$mRedis->zSet("DelayQs:job_zqueue", $expireTime, json_encode( $data ));
```

### 处理订单超时未付款 orderTimeOut.php
```php
<?php
	$body = trim( $_POST['body'] );
	$sign = trim( $_POST['sign'] );
	if ( !$body || !$sign ){
		echo "参数不对";
		retutn ;
	}
	//验证签名
	$data   = json_decode( $body, true );
	$privateKey = "sadhfusa7uy9iMn"; //签名加密密钥
	ksort($data);
	$mySign = md5( $privateKey.implode('&',$data) ); 
	if ( $mySign != $sign ){
		echo "sign签名错误对";
		retutn ;
	}
	
	//把$data['orderId']状态设置为关闭....
```

### test.go
```go
package main

import (
	"github.com/wyz9527/delayQS/tree/master/delayQS"
	"fmt"
)

func main(){
	if err := delayQS.Run(); err != nil{
		fmt.Println(err)
	}
	delayQS.Close()
}
```

### 编译
```shell
go build
```

### 已守护进程的方式 start
```shell
./test -opt=start -d=true
```

### stop
```shell
./test -opt=stop
```

### restart
```shell
./test -opt=restart
```







