# 文书网详情页采集
## 免责申明：请勿用于违法用途，用于违法用途产生的后果与本人无关，本版本仅供学习参考所用
## 项目特点
- 充分利用golang goroutine 的并发模型，减少代码量，便于理解，减轻编写并发异步代码负担。启动1000个协程访问，极大加快爬取速度，实测每日入库可达200W。
- 使用mongodb 数据库，利用mongodb FindOneAndUpdate 特性可实现分布式任务队列。
- 破解了js混淆加密
## 使用说明
- 项目需要已采集过docid列表
-  因私有代理不开源公布，项目需要自行实现getProxy 代理函数。
替换你的http代理API URL
```
func getProxy(){
	for {
		response, e := httpGet("替换你的http代理API",
			httpProxyClient, "")
		if e== nil {
			bytes, e := ioutil.ReadAll(response.Body)
			if e==nil{
				log.Println("获取200个代理")
				response.Body.Close()
				result := strings.Split(strings.Trim(string(bytes),"\r\n"), "\r\n")
				for _,v := range result {
					proxyPool<-v
				}
			}
		}
	}
}
```