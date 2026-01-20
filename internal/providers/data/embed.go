package data

import (
	_ "embed"
)

//go:embed garuda.json
var GarudaData []byte

//go:embed lionair.json
var LionAirData []byte

//go:embed batikair.json
var BatikAirData []byte

//go:embed airasia.json
var AirAsiaData []byte
