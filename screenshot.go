// Command screenshot is a chromedp example demonstrating how to take a
// screenshot of a specific element and of the entire browser viewport.
package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// var headers map[string]interface{} = map[string]interface{}{
// 	`cookie`: `wr_gid=x; wr_vid=x; wr_skey=x; wr_pf=x; wr_rt=x; wr_localvid=x; wr_name=x; wr_avatar=x; wr_gender=x`
// }

const cookie string = `wr_gid=x; wr_vid=x; wr_skey=x; wr_pf=x; wr_rt=x; wr_localvid=x; wr_name=x; wr_avatar=x; wr_gender=x`

const js_script = `
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
	dir, err := ioutil.TempDir("", "chromedp-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	opts := append(chromedp.DefaultExecAllocatorOptions[3:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	getBook(taskCtx, "https://weread.qq.com/web/reader/64e32bf071fd5a9164ece6bk65132ca01b6512bd43d90e3", cookie)

	// ret := screenshotPage(taskCtx)
	// file, _ := os.Create("result.png")
	// png.Encode(file, ret)

	fmt.Println("正常结束")
}

func getBook(ctx context.Context, url string, cookie string) {

	var evalbuf []byte
	// var strbuf string
	var pageCount int
	cookie_arr := strings.Split(cookie, ";")
	var cookies []string
	for _, cookie := range cookie_arr {
		cookietemp := strings.Split(cookie, "=")
		for _, cookieitem := range cookietemp {
			cookies = append(cookies, strings.TrimSpace(cookieitem))
		}
	}
	log.Println(cookies)

	// init
	if err := chromedp.Run(ctx,
		// network.Enable(),
		// network.SetExtraHTTPHeaders(network.Headers(headers)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			// add cookies to chrome
			for i := 0; i < len(cookies); i += 2 {
				log.Println(cookies[i] + "=" + cookies[i+1])
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).
					WithDomain(".qq.com").
					WithHTTPOnly(true).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`if (document.querySelector(".white")!=null){document.querySelector(".white").click()}`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerTopBar").style="display:none"`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerControls").style="display:none"`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerFooter_button").style="display:none"`, &evalbuf),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetAllCookies().Do(ctx)
			if err != nil {
				return err
			}

			for i, cookie := range cookies {
				log.Printf("chrome cookie %d: %+v", i, cookie)
			}

			return nil
		}),
	); err != nil {
		log.Fatal(err)
	}

	// get pageCount
	chromedp.Run(ctx,
		chromedp.Evaluate(`if (document.querySelector(".white")!=null){document.querySelector(".white").click()}`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerTopBar").style="display:none"`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerControls").style="display:none"`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerFooter_button").style="display:none"`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerCatalog_list").childElementCount`, &pageCount),
	)
	log.Println("page count is:", pageCount)

	// get page
	for i := 1; i < pageCount+1; i++ {
		fmt.Println(fmt.Sprint(`document.querySelector(".readerCatalog_list>li:nth-of-type(`, i, `)>div").click()`))
		// subctx, _ := context.WithCancel(ctx)
		// 选择章节并且初始化页面
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelector(".catalog").click()`, &evalbuf),
			chromedp.Click(fmt.Sprint(".readerCatalog_list>li:nth-of-type(", i, ")"), chromedp.NodeNotVisible),
			chromedp.Sleep(300*time.Millisecond),
			chromedp.Evaluate(`if(document.querySelector(".readerTopBar")!=null){document.querySelector(".readerTopBar").style="display:none"}`, &evalbuf),
			chromedp.Evaluate(`document.querySelector(".readerControls").style="display:none"`, &evalbuf),
			chromedp.Evaluate(`document.querySelector(".readerFooter_button").style="display:none"`, &evalbuf),
			// chromedp.ActionFunc(func(ctx context.Context) error {
			// 	ret := screenshotPage(ctx)
			// 	file, _ := os.Create(fmt.Sprint("book-", i, ".png"))
			// 	png.Encode(file, ret)
			// 	// cancel()
			// 	return nil
			// }),
		); err != nil {
			log.Fatal(err)
		}
		// 等待页面加载完成, 这里后续可以优化为 Dom 事件通知
		// time.Sleep(1 * time.Second)
		// 截图和保存
		ret := screenshotPage(ctx)
		file, _ := os.Create(fmt.Sprint("book-", i, ".png"))
		log.Println(fmt.Sprint("save book-", i, ".png"))
		png.Encode(file, ret)
	}
}

func screenshotPage(ctx context.Context) image.Image {

	var buf, evalbuf []byte
	var height int
	var boolbuf bool = false
	var scroll_height int = 0
	var imageFiles []string

	log.Println("开始截图...")
	dir, err := ioutil.TempDir("", "page")
	// defer os.RemoveAll(dir)
	fmt.Println(dir)
	if err != nil {
		log.Fatal(err)
	}

	for !boolbuf {
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(js_script, &boolbuf),
			chromedp.Evaluate(fmt.Sprint(`window.scrollTo(0,`, scroll_height, `)`), &evalbuf),
			chromedp.Evaluate(`document.body.clientHeight `, &height),
			chromedp.Screenshot(`.app_content`, &buf),
			chromedp.Sleep(100*time.Millisecond),
		); err != nil {
			log.Fatal(err)
		}

		scroll_height += height

		fmt.Println(fmt.Sprint(dir, "/", scroll_height, ".png"))
		imageFiles = append(imageFiles, fmt.Sprint(dir, "/", scroll_height, ".png"))
		if err := ioutil.WriteFile(fmt.Sprint(dir, "/", scroll_height, ".png"), buf, 0o644); err != nil {
			log.Fatal(err)
		}
	}

	log.Println(imageFiles)
	return mergeImages(imageFiles)
}

func mergeImages(imgs []string) image.Image {

	white := color.NRGBA{255, 255, 255, 255}

	newimgFile, err := os.Open(imgs[0])
	if err != nil {
		log.Fatalf("Failed to open %s", err)
	}
	newimg, _ := png.Decode(newimgFile)
	newRgba := image.NewNRGBA(newimg.Bounds())
	for w := 0; w < newimg.Bounds().Dx(); w++ {
		for h := 0; h < newimg.Bounds().Dy(); h++ {
			newRgba.SetNRGBA(w, h, white)
		}
	}

	for _, img := range imgs {
		imgItem, err := os.Open(img)
		if err != nil {
			log.Fatalf("Failed to open %s", err)
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
	if err != nil {
		log.Fatalf("Failed to decode %s", err)
	}
	return newRgba
}
