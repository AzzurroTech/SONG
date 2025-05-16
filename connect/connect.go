package connect

import (
	song "github.com/AzzurroTech/SONG"
)

type ConnectAPI struct {
	song.API
}

func InitAPI() *ConnectAPI {
	a := song.InitAPI()
	//make changes to a as required
	return a
}
