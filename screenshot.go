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
	"time"

	"github.com/chromedp/chromedp"
)

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

	initBookPage("https://weread.qq.com/web/reader/64e32bf071fd5a9164ece6bk65132ca01b6512bd43d90e3", taskCtx)
	ret := screenshotPage(taskCtx)
	file, _ := os.Create("result.png")
	png.Encode(file, ret)
	// 翻页逻辑
	// chromedp.Run(taskCtx,
	// 	chromedp.Click(".readerFooter_button", chromedp.NodeVisible),
	// )

	fmt.Println("正常结束")
}

func initBookPage(url string, ctx context.Context) {
	var evalbuf []byte
	chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.querySelector(".white").click()`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerTopBar").remove()`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerControls").remove()`, &evalbuf),
		chromedp.Evaluate(`document.querySelector(".readerFooter_button").style="display:none"`, &evalbuf),
	)
}

func screenshotPage(ctx context.Context) image.Image {

	var buf, evalbuf []byte
	var height int
	var boolbuf bool = false
	var scroll_height int = 0
	var imageFiles []string

	dir, err := ioutil.TempDir("", "page")
	fmt.Println(dir)
	if err != nil {
		log.Fatal(err)
	}

	for !boolbuf {
		chromedp.Run(ctx,
			chromedp.Evaluate(js_script, &boolbuf),
			chromedp.Evaluate(fmt.Sprint(`window.scrollTo(0,`, scroll_height, `)`), &evalbuf),
			chromedp.Evaluate(`document.body.clientHeight `, &height),
			chromedp.Sleep(100*time.Millisecond),
		)

		scroll_height += height

		if err := chromedp.Run(
			ctx,
			chromedp.Evaluate(fmt.Sprint(`window.scrollTo(0,`, scroll_height, `)`), &evalbuf),
			chromedp.Screenshot(`.app_content`, &buf),
		); err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprint(dir, "-", scroll_height, ".png"))
		imageFiles = append(imageFiles, fmt.Sprint(dir, "-", scroll_height, ".png"))
		if err := ioutil.WriteFile(fmt.Sprint(dir, "-", scroll_height, ".png"), buf, 0o644); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(imageFiles)
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
