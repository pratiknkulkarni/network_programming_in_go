package zero

import "time"

func Zero() {
	go Server()
	time.Sleep(time.Millisecond * 200)
	Client()
}
