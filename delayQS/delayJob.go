package delayQS

type delayJob struct {
	Id string
	Body interface{} `json:"body"` //任务的数据 json格式
	Callback string `json:"callback"` //回调地址 url
	Sign string `json:"sign"` //验证字符串
}



