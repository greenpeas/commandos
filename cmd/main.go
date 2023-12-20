package main

import (
	"commandos/internal/app"
)

const confPath string = "configs/app.yml"

func main() {

	app.InitAndRun(confPath)
}
