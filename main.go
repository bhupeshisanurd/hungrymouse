package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

// delay is a helper function that halts the program for a given amount of time
func delay(ctx context.Context, time time.Duration) {
	err := chromedp.Run(ctx, chromedp.Sleep(time))
	if err != nil {
		log.Fatal(err)
	}
}

func GetMousePosition(ctx context.Context) (float64, float64, error) {
	// Execute JavaScript to get coordinates of image
	var imageInfo map[string]float64

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`(function() {
			const canvasRect = canvas.getBoundingClientRect();

			var imageCenterX = canvasRect.left + position.x + imgWidth / 2;
			var imageCenterY = canvasRect.top + position.y + imgHeight / 2;

			return { imageCenterX, imageCenterY };
        })();`, &imageInfo),
	); err != nil {
		return 0, 0, err
	}

	x := imageInfo["imageCenterX"]
	y := imageInfo["imageCenterY"]

	return x, y, nil
}

func DragElement(ctx context.Context, initialX, initialY, finalX, finalY, factor float64) error {
	// A drag consists of 3 events: mouse pressed, mouse moved, mouse released
	p := &input.DispatchMouseEventParams{
		Type:       input.MousePressed,
		X:          initialX,
		Y:          initialY,
		Button:     input.Left,
		ClickCount: 1,
	}
	c := chromedp.FromContext(ctx)

	if err := p.Do(cdp.WithExecutor(ctx, c.Target)); err != nil {
		return errors.Wrap(err, "could not do left-click on mouse")
	}

	// Mouse Move
	p.Type = input.MouseMoved
	if finalX > 0 {
		p.X = initialX + factor
	}
	if finalY > 0 {
		p.Y = initialY + factor
	}

	if err := p.Do(cdp.WithExecutor(ctx, c.Target)); err != nil {
		return errors.Wrap(err, "could not move mouse")
	}

	p.Type = input.MouseReleased
	if err := p.Do(cdp.WithExecutor(ctx, c.Target)); err != nil {
		return errors.Wrap(err, "could not release mouse")
	}

	return nil
}

func StartAutomation(dir string) error {
	// take url from command line
	if len(os.Args) < 2 {
		fmt.Println("Please provide url")
		os.Exit(1)
	}

	// set up a new Chrome instance
	dir, err := os.MkdirTemp("", dir)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", false),
		chromedp.Flag("window-size", "1920,1080"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	log.Println("Starting Chrome")
	// ensure that the browser process is started
	if err := chromedp.Run(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Opening Hungry Mouse")
	url := os.Args[1]
	err = chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		return err
	}
	delay(ctx, 2*time.Second)

	var times int = 10
	var factor float64 = 90

	for i := 0; i < times; i++ {
		// Get Mouse Position
		x, y, err := GetMousePosition(ctx)
		if err != nil {
			return err
		}

		// Drag Element
		err = DragElement(ctx, x, y, x, 0, factor)
		if err != nil {
			return err
		}
	}

	delay(ctx, 5*time.Second)
	return nil
}

func main() {
	// start 2 instances of Chrome using goroutines

	var wg sync.WaitGroup

	// 2 goroutines
	wg.Add(1)
	go func(dir string) {
		defer wg.Done()
		err := StartAutomation(dir)
		if err != nil {
			log.Fatal(err)
		}
	}("dir1")
	// go func(dir string) {
	// 	defer wg.Done()
	// 	StartAutomation(dir)
	// }("dir2")

	wg.Wait()

}
