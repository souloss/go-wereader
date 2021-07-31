// Command screenshot is a chromedp example demonstrating how to take a
// screenshot of a specific element and of the entire browser viewport.
package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// 设置 headers 的方式在这一层次不好使...
// var headers map[string]interface{} = map[string]interface{}{
// 	`cookie`: `wr_gid=x; wr_vid=x; wr_skey=x; wr_pf=x; wr_rt=x; wr_localvid=x; wr_name=x; wr_avatar=x; wr_gender=x`
// }
var cookie string = os.Args[1]

const wereaderCategoryUrl string = `https://weread.qq.com/web/category/`
const wereaderUrl string = `https://weread.qq.com/`

const isScrollTailScript = `
function getScrollTop()
{
　　var scrollTop = 0, bodyScrollTop = 0, documentScrollTop = 0;
　　if(document.body){
　　　　bodyScrollTop = document.body.scrollTop;
　　}
　　if(document.documentElement){
　　　　documentScrollTop = document.documentElement.scrollTop;
　　}
scrollTop = (bodyScrollTop - documentScrollTop > 0) ? bodyScrollTop : documentScrollTop;
return scrollTop;
}
function getScrollHeight(){
　　var scrollHeight = 0, bodyScrollHeight = 0, documentScrollHeight = 0;
　　if(document.body){
　　　　bSH = document.body.scrollHeight;
　　}
　　if(document.documentElement){
　　　　dSH = document.documentElement.scrollHeight;
　　}
scrollHeight = (bSH - dSH > 0) ? bSH : dSH ;
　　return scrollHeight;
}
function getWindowHeight(){
　　var windowHeight = 0;
　　if(document.compatMode == "CSS1Compat"){
　　　　windowHeight = document.documentElement.clientHeight;
　　}else{
　　　　windowHeight = document.body.clientHeight;
　　}
　　return windowHeight;
}
getScrollTop() + getWindowHeight() == getScrollHeight()
`

func main() {

	// get custom opts and ctx
	_, taskCtx, cancel1, cancel2 := NewBrowerCtx(true)
	defer cancel1()
	defer cancel2()

	cookies := cookiesStrToArr(cookie)
	log.Println(cookies)
	setCookies(taskCtx, wereaderUrl, cookies)
	// category := getCategory(taskCtx)
	// fmt.Println(category)

	const parallerCount int = 5
	idle := make(chan struct{}, parallerCount)
	tasks := make(chan string, parallerCount)
	ok := make(chan struct{})
	go producer(taskCtx, tasks, ok)

	for w := 1; w <= parallerCount; w++ {
		idle <- struct{}{}
		go worker(w, tasks, idle)
	}

	// 等待生产者完成
	<-ok
	// 等待消费者停止工作
	for i := 1; i <= parallerCount; i++ {
		<-idle
	}

	// getBook(taskCtx, "https://weread.qq.com/web/reader/ecb325e05cfecbecb37efec", cookie)

	log.Println("Perfect Ending !")
}

func worker(id int, tasks chan string, idle chan struct{}) {
	for j := range tasks {
		<-idle
		log.Println("worker", id, "started  job", j)
		_, taskCtx, cancel1, cancel2 := NewBrowerCtx(true)
		GetBook(taskCtx, j, cookie)
		cancel1()
		cancel2()
		log.Println("worker", id, "finished job", j)
		idle <- struct{}{}
	}
}

func producer(ctx context.Context, tasks chan string, ok chan struct{}) {
	bookUrlsFromCategory := getBookUrlsFromCategory(ctx, "计算机榜")
	for key, value := range bookUrlsFromCategory {
		log.Println("开始生产 ", key, "类别的图书!")
		for ikey, ivalue := range value {
			log.Println("send", ikey, ",", ivalue, " => workerPools")
			tasks <- ivalue
		}
	}
	ok <- struct{}{}
}

func getCategory(ctx context.Context) map[string]string {
	var category map[string]string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(wereaderCategoryUrl),
		chromedp.Evaluate(`q={};document.querySelectorAll(".ranking_list>li>a").forEach(a=>{q[a.text.replace(/\s*/g,"")]=a.href});q`, &category),
	); err != nil {
		log.Panic(err)
	}
	return category
}

// 从分类获取书本的url
// 返回值格式类似 {子分类1:{书本名1:url1},子分类2:{书本2:url2}}
func getBookUrlsFromCategory(ctx context.Context, category string) map[string]map[string]string {
	var evalbuf []byte
	var subCategoryBookUrls map[string]map[string]string = make(map[string]map[string]string)
	var categoryUrl string
	var subCategoryCount int
	var bookUrlsCount int

	categoryUrl, isOk := getCategory(ctx)[category]
	if !isOk {
		log.Panic("category", category, "not found!")
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(categoryUrl),
		chromedp.Evaluate(`document.querySelector(".ranking_page_header_categroy_container").childElementCount`, &subCategoryCount),
	); err != nil {
		log.Panic(err)
	}
	log.Println(subCategoryCount, "book URLs in the", category, "category")

	// 遍历获取每个子类别中的图书URL
	for i := 1; i < subCategoryCount+1; i++ {
		var subCategoryName string
		log.Println(fmt.Sprint(`document.querySelector(".ranking_page_header_categroy_container>div:nth-of-type(`, i, `)").click()`))
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprint(`document.querySelector(".ranking_page_header_categroy_container>div:nth-of-type(`, i, `)").textContent`), &subCategoryName),
			chromedp.Evaluate(fmt.Sprint(`document.querySelector(".ranking_page_header_categroy_container>div:nth-of-type(`, i, `)").click()`), &evalbuf),
		); err != nil {
			log.Panic(err)
		}
		// 在这个子类别下一直翻页获取个数
		tempBookUrlsCount := 0
		retryCount := 0
		for {
			tempBookUrlsCount = bookUrlsCount
			if err := chromedp.Run(ctx,
				// 每次滑动的长度不一样更容易触发异步加载
				chromedp.Evaluate(fmt.Sprint(`window.scrollTo(0,`, bookUrlsCount*10000000000, `)`), &evalbuf),
				chromedp.Query(`.ranking_content_bookList`, chromedp.NodeVisible),
				chromedp.Evaluate(`document.querySelector(".ranking_content_bookList").childElementCount `, &bookUrlsCount),
				// 这里必须等待书本项进行异步加载
				chromedp.Sleep(200*time.Millisecond),
			); err != nil {
				log.Panic(err)
			}
			log.Println(fmt.Sprint("目前异步加载的书本数量", bookUrlsCount, "; 上次异步加载的书本数量", tempBookUrlsCount))
			if bookUrlsCount == tempBookUrlsCount {
				retryCount++
				if retryCount == 3 {
					break
				}
			}
		}
		var bookUrls map[string]string
		// 翻页直到动态加载完所有 li 后，获取这个子类别所有图书的url
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(`m={};document.querySelectorAll(".ranking_content_bookList>li").forEach(q=>{m[q.querySelector('.wr_bookList_item_title').textContent]=q.querySelector('a').href});m`, &bookUrls),
			chromedp.Evaluate(`document.querySelector(".ranking_content_bookList").childElementCount `, &bookUrlsCount),
			chromedp.Sleep(300*time.Millisecond),
		); err != nil {
			log.Panic(err)
		}
		log.Println(subCategoryName, "子类收集完成数量为:", len(bookUrls))
		log.Println(bookUrls)
		subCategoryBookUrls[subCategoryName] = bookUrls
	}

	return subCategoryBookUrls
}

// 获取一个浏览器实例和标签页的实例的 Context 和 Cancel
func NewBrowerCtx(headless bool) (context.Context, context.Context, context.CancelFunc, context.CancelFunc) {
	defaultOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
	)
	opts := []chromedp.ExecAllocatorOption{}
	if !headless {
		for _, opt := range defaultOpts {
			if reflect.ValueOf(opt).Pointer() != reflect.ValueOf(chromedp.Headless).Pointer() {
				opts = append(opts, opt)
			}
		}
	} else {
		opts = defaultOpts
	}
	allocCtx, callocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	tabCtx, tabCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	return allocCtx, tabCtx, callocCancel, tabCancel
}

// 将 cookies 字符串转换为形如 [key1,value1,key2,value2,key3,value3] 的 cookie 数组
func cookiesStrToArr(cookie string) []string {
	cookie_arr := strings.Split(cookie, ";")
	var cookies []string
	for _, cookie := range cookie_arr {
		cookietemp := strings.Split(cookie, "=")
		for _, cookieitem := range cookietemp {
			cookies = append(cookies, strings.TrimSpace(cookieitem))
		}
	}
	return cookies
}

// 获取 url 的 domain
// url format: protocal://domain:port/path/
// func getUrlDomain(url string) (string, error) {
// 	urlRegexp := regexp.MustCompile(`^https?://([\w.]*(:\d+)?)//?.*`)
// 	subString := urlRegexp.FindStringSubmatch(url)
// 	if len(subString) > 1 {
// 		return subString[1], nil
// 	} else {
// 		return "", errors.New(fmt.Sprint("url ", url, "not has domain!"))
// 	}
// }

// 在浏览器标签页中为url设置cookies
// 注意,这个方法会将这个标签页实例的导航栏切换到这个url
func setCookies(ctx context.Context, urlStr string, cookies []string) {
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Panic(err)
	}
	if err := chromedp.Run(ctx,
		// network.Enable(),
		// network.SetExtraHTTPHeaders(network.Headers(headers)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			// add cookies to chrome
			for i := 0; i < len(cookies); i += 2 {
				// log.Println(cookies[i] + "=" + cookies[i+1])
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).
					WithDomain(u.Host).
					WithHTTPOnly(true).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		chromedp.Navigate(urlStr),
		// chromedp.ActionFunc(func(ctx context.Context) error {
		// 	cookies, err := network.GetAllCookies().Do(ctx)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	for i, cookie := range cookies {
		// 		log.Printf("chrome cookie %d: %+v", i, cookie)
		// 	}
		// 	return nil
		// }),
	); err != nil {
		log.Panic(err)
	}
}

// 获取该书籍元信息
func getBookMeta(ctx context.Context) (string, string, string, int, bool) {
	var title string
	var keywords string
	var description string
	var pageCount int
	var lock bool
	var ok bool

	// get pageCount
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector(".readerCatalog_list").childElementCount`, &pageCount),
		chromedp.Title(&title),
		chromedp.AttributeValue(`meta[name=keywords]`, `content`, &keywords, &ok),
		chromedp.AttributeValue(`meta[name=description]`, `content`, &description, &ok),
		chromedp.Evaluate(`a=false;if(document.querySelector(".chapterItem_lock")){a=true};a`, &lock),
	); err != nil {
		// 发生错误就当做未解锁
		lock = true
		return title, keywords, description, pageCount, lock
	}
	log.Println("page metainfo is:", title, keywords, description, pageCount, lock)
	return title, keywords, description, pageCount, lock
}

// 获取书籍
func GetBook(ctx context.Context, url string, cookie string) error {

	var evalbuf []byte
	// .chapterItem_lock 判断
	var retry int = 0
	const retryMax int = 3

	cookies := cookiesStrToArr(cookie)

	// init
	setCookies(ctx, url, cookies)

	// get Book metainfo
	title, keywords, description, pageCount, lock := getBookMeta(ctx)
	if lock {
		log.Println(`书籍《`, title, `》还未解锁，进行跳过!`)
		return nil
	}
	os.MkdirAll(title, os.ModePerm)
	metafile, _ := os.Create(fmt.Sprint(title, `/`, `meta.txt`))
	defer metafile.Close()
	metafile.WriteString(fmt.Sprint(`{"title":"`, title, `","keywords":"`, keywords, `","description":"`, description, `"}`))

	// get page
	for i := 1; i < pageCount+1; i++ {

		if _, err := os.Stat(fmt.Sprint(title, `/`, i, `.png`)); err == nil {
			log.Println(`书籍 《`, title, `》 第`, i, `页已存在，进行跳过!`)
			continue
		} else {
			log.Println(`开始书籍《`, title, `》第`, i, `页的截取`)
		}
		// 选择章节并且初始化页面
	savepage:
		if err := chromedp.Run(ctx,
			// chromedp.Query(`.catalog`, chromedp.NodeVisible),
			chromedp.Evaluate(`document.querySelector(".catalog").click()`, &evalbuf),
			chromedp.Click(fmt.Sprint(".readerCatalog_list>li:nth-of-type(", i, ")"), chromedp.NodeNotVisible),
			chromedp.Sleep(900*time.Millisecond),
			chromedp.Query(`.app_content`, chromedp.NodeVisible),
			chromedp.Evaluate(`if (document.querySelector(".white")!=null){document.querySelector(".white").click()}`, &evalbuf),
			chromedp.Evaluate(`if(document.querySelector(".readerTopBar")!=null){document.querySelector(".readerTopBar").style="display:none"}`, &evalbuf),
			chromedp.Evaluate(`if(document.querySelector(".readerControls")!=null){document.querySelector(".readerControls").style="display:none"}`, &evalbuf),
			chromedp.Evaluate(`if(document.querySelector(".readerFooter_button")!=null){document.querySelector(".readerFooter_button").style="display:none"}`, &evalbuf),
		); err != nil {
			retry++
			if retry >= retryMax {
				return errors.New("选择章节超过最大重试次数")
			} else {
				goto savepage
			}
		}
		retry = 0
		// 截图和保存
		ret, err := screenshotPage(ctx)
		if err != nil {
			return err
		}
		file, _ := os.Create(fmt.Sprint(title, `/`, i, `.png`))
		png.Encode(file, ret)
		file.Close()
	}
	return nil
}

// 以页面高度(document.body.clientHeight)为单位滑动滑条并截图存放到临时目录，每次等待 500ms 加载内容，最后将这些图片覆盖合并并返回
func screenshotPage(ctx context.Context) (image.Image, error) {

	var buf, evalbuf []byte
	var height int
	var boolbuf bool = false
	var scroll_height int = 0
	var imageFiles []string
	const retryMax int = 3
	var retry int

	log.Println("开始截图...")
	dir, err := ioutil.TempDir("", "page")
	defer os.RemoveAll(dir)
	if err != nil {
		return nil, err
	} else {
		log.Println("创建页面临时目录: ", dir)
	}

	// page screenshot and save
	for !boolbuf {
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(isScrollTailScript, &boolbuf),
			chromedp.Evaluate(fmt.Sprint(`window.scrollTo(0,`, scroll_height, `)`), &evalbuf),
			chromedp.Evaluate(`document.body.clientHeight `, &height),
			chromedp.Screenshot(`.app_content`, &buf),
			chromedp.Sleep(500*time.Millisecond),
		); err != nil {
			retry++
			if retry >= retryMax {
				return nil, errors.New("the screenshot of the page exceeds the maximum number of retries")
			}
			log.Println(err)
			continue
		}

		if err := ioutil.WriteFile(fmt.Sprint(dir, "/", scroll_height, ".png"), buf, 0o644); err != nil {
			retry++
			if retry >= retryMax {
				return nil, errors.New("the save of the page exceeds the maximum number of retries")
			}
			log.Println(err)
		} else {
			retry = 0
			imageFiles = append(imageFiles, fmt.Sprint(dir, "/", scroll_height, ".png"))
			scroll_height += height
		}
	}

	image, err := mergeImages(imageFiles)
	if err != nil {
		// 这里可以处理合并错误的问题，再继续重新合并
		return nil, err
	}
	return image, nil
}

// 图片合并，填充所有白色区域
func mergeImages(imgs []string) (image.Image, error) {

	// 白色的 NRGBA 表示
	white := color.NRGBA{255, 255, 255, 255}
	// 以第一张图片的大小为基准创建纯白色的 NRGBA 图片
	baseImg, err := os.Open(imgs[0])
	if err != nil {
		return nil, err
	}
	newimg, err := png.Decode(baseImg)
	if err != nil {
		return nil, err
	}
	newRgba := image.NewNRGBA(newimg.Bounds())
	for w := 0; w < newimg.Bounds().Dx(); w++ {
		for h := 0; h < newimg.Bounds().Dy(); h++ {
			newRgba.SetNRGBA(w, h, white)
		}
	}

	// 遍历每张图片的每个点，若新创建的 NRGBA 图的该点为空白则进行填充
	for _, img := range imgs {
		imgItem, err := os.Open(img)
		if err != nil {
			return nil, err
		}
		decodedItem, _ := png.Decode(imgItem)
		for w := 0; w < newimg.Bounds().Dx(); w++ {
			for h := 0; h < newimg.Bounds().Dy(); h++ {
				if newRgba.At(w, h) == white {
					r, g, b, a := decodedItem.At(w, h).RGBA()
					newRgba.SetNRGBA(w, h, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
				} else {
					var r, g, b, a = newRgba.At(w, h).RGBA()
					newRgba.SetNRGBA(w, h, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
				}
			}
		}
		defer imgItem.Close()
	}

	return newRgba, nil
}
