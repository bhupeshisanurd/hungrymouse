package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
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

			var imageCenterX = canvasRect.left + position.x + mouseWidth / 2;
			var imageCenterY = canvasRect.top + position.y + mouseHeight / 2;

			return { imageCenterX, imageCenterY };
        })();`, &imageInfo),
	); err != nil {
		return 0, 0, err
	}

	x := imageInfo["imageCenterX"]
	y := imageInfo["imageCenterY"]

	return x, y, nil
}

func GetCheesePosition(ctx context.Context) (float64, float64, error) {
	// Execute JavaScript to get coordinates of cheese
	var imageInfo map[string]float64

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`(function() {
			const canvasRect = canvas.getBoundingClientRect();

			var imageCenterX = canvasRect.left + cheesePosition.x + cheeseWidth / 2;
			var imageCenterY = canvasRect.top + cheesePosition.y + cheeseHeight / 2;

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

	p.X = finalX
	p.Y = finalY
	steps := int(math.Max(math.Abs(finalX-initialX), math.Abs(finalY-initialY)) / factor)

	// Mouse Move
	p.Type = input.MouseMoved
	for i := 1; i <= steps; i++ {
		p.X = initialX + (finalX-initialX)*float64(i)/float64(steps)
		p.Y = initialY + (finalY-initialY)*float64(i)/float64(steps)

		if err := p.Do(cdp.WithExecutor(ctx, c.Target)); err != nil {
			return errors.Wrap(err, "could not move mouse")
		}

		// Add a delay to make the movement smoother
		time.Sleep(time.Millisecond * 80)
	}

	p.Type = input.MouseReleased
	if err := p.Do(cdp.WithExecutor(ctx, c.Target)); err != nil {
		return errors.Wrap(err, "could not release mouse")
	}

	return nil
}

func IsCheeseFound(ctx context.Context) (bool, error) {
	// find the cheese by getting it by "cheese-image" id
	var cheeseFound bool = false
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`(function() {
			const cheese = document.getElementById("cheese-image");
			return cheese !== null;
		})();`, &cheeseFound),
	); err != nil {
		return false, err
	}

	return cheeseFound, nil
}

func StartAutomation() error {
	// take url from command line
	if len(os.Args) < 2 {
		fmt.Println("Please provide url")
		os.Exit(1)
	}

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
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
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		return err
	}
	delay(ctx, 2*time.Second)

	// Find the email input field by CSS selector
	inputSelector := `input#cheese-count`
	err = chromedp.Run(ctx, chromedp.Focus(inputSelector))
	if err != nil {
		return err
	}

	fmt.Println("Filling in cheese slice count")
	// Fill in the email field
	var cheeseCount string = "5"
	err = chromedp.Run(ctx, chromedp.SendKeys(inputSelector, cheeseCount))
	if err != nil {
		return err
	}

	fmt.Println("Clicking Let me eat button")
	err = chromedp.Run(ctx, chromedp.Click(`#submit-button`))
	if err != nil {
		return err
	}

	// var times int = 10
	var factor float64 = 50

	for {
		cheeseFound, err := IsCheeseFound(ctx)
		if err != nil {
			return err
		}

		if !cheeseFound {
			fmt.Println("I am full 🐭, no more cheese")
			break
		}

		// Get Mouse Position
		mouseX, mouseY, err := GetMousePosition(ctx)
		if err != nil {
			return err
		}

		// Get Cheese Position
		cheeseX, cheeseY, err := GetCheesePosition(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("There is cheese at x:%v, y:%v\n", cheeseX, cheeseY)

		// Drag Element
		err = DragElement(ctx, mouseX, mouseY, cheeseX, cheeseY, factor)
		if err != nil {
			return err
		}

		fmt.Println("There is still cheese 🧀 left")
	}

	delay(ctx, 3*time.Second)
	return nil
}

func main() {
	err := StartAutomation()
	if err != nil {
		log.Fatal(err)
	}
}
