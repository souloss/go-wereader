package main

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestParaller(t *testing.T) {
	// 若卡死可以在管理员控制台下执行 taskkill /f /im chrome.exe 强制关闭进程
	var paraller int = 10
	wg := sync.WaitGroup{}

	wg.Add(paraller)
	// 控制创建进程的速度
	// time.Sleep(time.Duration(paraller) * time.Second)
	for i := 0; i < paraller; i++ {
		go func(i int) {
			// 这里可以控制是否以无头模式运行
			_, tabCtx, cancel1, cancel2 := NewBrowerCtx(false)
			defer cancel1()
			defer cancel2()
			chromedp.Run(
				tabCtx,
				chromedp.Navigate(wereaderUrl),
				chromedp.Sleep(5*time.Second),
			)
			wg.Done()
		}(i)
	}
	wg.Wait()

	log.Println("Perfect Ending !")
}
