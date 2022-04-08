package mtgsdk

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var (
	browser *rod.Browser = nil // The browser that accesses the edhrec website

)

// Initializes the browser
func initBrowser() error {
	if browser != nil {
		return nil
	}
	mutex.Lock()
	u := launcher.New().
		Set("--blink-settings=imagesEnabled=false").
		MustLaunch()

	browser = rod.New().ControlURL(u)
	err := browser.Connect()
	if err != nil {
		mutex.Unlock()
		return err
	}
	logger.Println("Connected to browser")
	mutex.Unlock()
	return nil
}

// Navigates the browser to the specified url
func nav(url string) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}
	waitFunc := page.MustWaitNavigation()
	err = page.Navigate(url)
	if err != nil {
		return nil, err
	}
	logger.Println("Connected to the page, rendering...")
	waitFunc()
	logger.Println("Page rendered!")
	return page, nil
}
