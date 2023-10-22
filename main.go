package main

import (
	"fmt"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
	// "github.com/ka2n/go-idle"
)

func main() {
	fmt.Println("Hello!")
	var err error
	var isIdle bool
	// idle.Get()
	for err == nil {
		isIdle, err = isInactive()
		// get current time and print it
		fmt.Printf("%v:  ", time.Now())
		if isIdle {
			fmt.Println("Screen is off")
		} else {
			fmt.Println("Screen is on")
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		fmt.Println(err)
	}
}

func isInactive() (bool, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	info := xproto.Setup(conn)
	screen := info.DefaultScreen(conn)

	if err := screensaver.Init(conn); err != nil {
		return false, err
	}

	rep, err := screensaver.QueryInfo(conn, xproto.Drawable(screen.Root)).Reply()
	if err != nil {
		return false, err
	}

	return rep.State == 1, nil
}
