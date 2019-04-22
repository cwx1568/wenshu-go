# 文书网详情页采集
## 免责申明：请勿用于违法用途，用于违法用途产生的后果与本人无关，本版本仅供学习参考所用
## 说明
- 项目需要已采集过docid列表，利用mongodb FindOneAndUpdate 特性可实现分布式任务队列
-  因私有代理不开源公布，项目需要自行实现getProxy 代理函数。

```
func getProxy(){
	for {
		response, e := httpGet("替换你的http代理API URL",
			httpProxyClient, "")
		if e== nil {
			bytes, _ := ioutil.ReadAll(response.Body)
			log.Println("获取200个代理")
			response.Body.Close()
			result := strings.Split(strings.Trim(string(bytes),"\r\n"), "\r\n")
			for _,v := range result {
				proxyPool<-v
			}
		}
	}
}
```