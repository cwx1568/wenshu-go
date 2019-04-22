package main

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 配置日志
func init(){
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
// mongodb docid collection
var collection = func() *mongo.Collection {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://192.168.1.171:30000"))
	if err != nil {
		log.Println(err)
	}
	collection := client.Database("wenshu").Collection("wenshu_2018")
	return collection
}()

// 获取任务函数
func getTask(finish chan bool, tasks chan string) {
	t := time.Now().Unix()
	for {
		result := bson.M{}
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		_ = collection.FindOneAndUpdate(ctx, bson.M{"HasHtml": nil, "$or": bson.A{bson.M{"Processing": nil}, bson.M{"Processing": bson.M{"$lte": t - 1}}}}, bson.M{"$set": bson.M{"Processing": t}}).Decode(&result)
		if result["_id"] == nil {
			break
		} else {
			s := result["_id"].(string)
			tasks <- s
		}
	}
	close(tasks)
	finish <- true
}

// 请求详情
func httpCreateContentJS(client *http.Client, docId string, cookie *string) string {
	for {
		url2 := "http://wenshu.court.gov.cn/CreateContentJS/CreateContentJS.aspx?DocID=" + docId
		//log.Println("url2:"+url2)
		response1, e := httpGet(url2, client, *cookie)
		if e == nil {
			bytes, _ := ioutil.ReadAll(response1.Body)
			response1.Body.Close()
			result := string(bytes)
			if strings.Contains(result, "请开启JavaScript并刷新该页") {
				*cookie = httpIndex(client)
				if *cookie==""{
					changeProxy(client)
				}
			} else if strings.Contains(result, "//初始化全文插件") || strings.Contains(result, "此篇文书不存在") {
				log.Println("更新docid:"+docId)
				log.Println(result)
				b := true
				ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
				_, e := collection.UpdateOne(ctx, bson.M{"_id": docId}, bson.M{"$set": bson.M{"html": result, "HasHtml": true}}, &options.UpdateOptions{Upsert: &b})
				if e!= nil {
					log.Println(e)
				} else{
					return result
				}
			} else {
				log.Println("=========================================:"+result)
				log.Println("重试")
				changeProxy(client)
			}
		} else {
			changeProxy(client)
			log.Println(e)
		}
	}
}
var httpProxyClient = &http.Client{}

// 获取代理
func changeProxy(client *http.Client) {
	s:=<-proxyPool
	proxyUrl, _ := url.Parse("http://"+s)
	client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
}


// 解密js，获取cookie
func httpIndex(client *http.Client) string {
	resp, err := httpGet("http://wenshu.court.gov.cn/content/content?DocID=eff7f53c-b647-11e3-84e9-5cf3fc0c2c18&KeyWord=", client, "")
	if err != nil {
		log.Println(err)
	}else{
		document, err1 := goquery.NewDocumentFromReader(resp.Body)
		if err1 == nil {
			setCookie := strings.Split(resp.Header.Get("Set-Cookie"), "path=/")[0]
			defer resp.Body.Close()
			find := document.Find("[type=\"text/javascript\"]")
			text := find.Text()
			compile := regexp.MustCompile(`[\s\S]*(eval\(.+[\s\S]*\{\}\)\))[\s\S]*`)
			match := compile.FindStringSubmatch(text)
			vm := otto.New()
			if len(match)> 1 {
				if value, err := vm.Run(match[1] + decryptJs); err == nil  {
					url1 := "http://wenshu.court.gov.cn" + value.String()
					response, err := httpGet(url1, client, setCookie)
					if err != nil {
						log.Println(err)
					}else{
						defer response.Body.Close()
						bytes, _ := ioutil.ReadAll(response.Body)
						log.Println(string(bytes))
						setCookie1 := strings.Split(response.Header.Get("Set-Cookie"), "path=/")[0]
						return setCookie1
					}
				} else {
					log.Println(err)
				}
			}
		}else{
			log.Println(err1)
		}
	}
	return ""
}

func httpGet(url string, client *http.Client, cookie string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Referer", "http://wenshu.court.gov.cn/content/content?DocID=eff7f53c-b647-11e3-84e9-5cf3fc0c2c18&KeyWord=")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36")
	return client.Do(req)
}
// 处理任务队列
func processTask(tasks chan string, process chan struct{}) {
	cookie := ""
	client := &http.Client{
		//Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	changeProxy(client)
	for docId := range tasks {
		println(docId)
		httpCreateContentJS(client, docId, &cookie)
	}
	process <- struct{}{}
}
// 解密js，获取二级跳转url
var decryptJs = func() string {
	buf, _ := ioutil.ReadFile("decrypt.js")
	return string(buf)
}()

var proxyPool=make(chan string,200)
func main() {
	//vm := otto.New()
	finish := make(chan bool)

	process := make(chan struct{})
	tasks := make(chan string, 1000)
	routineCount := 1000

	//获取任务
	go getTask(finish, tasks)
	//获取代理
	go getProxy()

	//处理任务
	for i := 0; i < routineCount; i++ {
		go processTask(tasks, process)
	}

	// 确保routine都结束
	for i := 0; i < routineCount; i++ {
		<-process
	}
	<-finish


	s:=""
	fmt.Scanf("%s", &s)
}
