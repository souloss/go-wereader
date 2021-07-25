import time
from selenium.webdriver import Chrome, ChromeOptions

chrome_options = ChromeOptions()
chrome_options.add_argument('--headless')
chrome_options.add_argument('lang=zh_CN.UTF-8')
chrome_options.add_argument('--disable-gpu')
# chrome_options.add_argument('user-agent="MQQBrowser/26 Mozilla/5.0 (Linux; U; Android 2.3.7; zh-cn; MB200 Build/GRJ22; CyanogenMod-7) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1"')
# chrome_options.add_argument('user-agent="Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1"')
js_script="""
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
return getScrollTop() + getWindowHeight() == getScrollHeight()
"""
browser  = Chrome(options=chrome_options)
browser.get("https://weread.qq.com/web/reader/64e32bf071fd5a9164ece6bk65132ca01b6512bd43d90e3")
time.sleep(3)
browser.execute_script("document.querySelector(\".white\").click()")
y=0
print(y)
while not browser.execute_script(js_script):
    browser.execute_script(f"window.scrollTo(0,{y})")
    y=y+1
    print(y)
    print(browser.execute_script(js_script))
browser.execute_script("document.querySelector(\".readerTopBar\").remove()")
app_content = browser.find_element_by_class_name("app_content")
app_content.screenshot("png.png")
browser.get_screenshot_as_file('png_full.png')

browser.execute_cdp_cmd("Page.captureScreenshot","""
{
    "format": "png",
    "quality": 100,
    "clip": {
        "x": 388,
        "y": 722.765625,
        "width": 410,
        "height": 50,
        "scale": 1
    },
    "fromSurface": true,
    "captureBeyondViewport": true
}
""")