package main

import (
	"privacy-profile-composer/composer"
	"time"
)

func main() {
	go composer.Run_server()
	time.Sleep(time.Second)
	composer.Run_client()
}
