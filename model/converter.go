package model

type CanConverter struct {
	SN string
	No uint8

	// 2: relay, 3: 485
	Code int

	Tx chan<- []byte
}
type LoraConverter struct {
	SN string

	Id []byte

	// io:0x04, 485:0x05
	Cmd byte

	// 4: io, 5: 485
	LoraType int

	Tx chan<- []byte
}
type SerialConverter struct {
	SN string

	Addr uint8

	Tx chan<- []byte
}

func sendData(data []byte, tx chan<- []byte) {
	tx <- data
}

func (c *CanConverter) Setting485(speed uint32, event byte, data byte, stop byte) {
	d := []byte{0x01, byte(speed), byte(speed >> 8), byte(speed >> 16), byte(speed >> 24), data, event, stop}
	// c.Tx <- combineCanData(c.No, 5, d)
	go sendData(combineCanData(c.No, 5, d), c.Tx)
}

func combineCanData(no uint8, code int, data []byte) []byte {
	len := 2 + len(data)

	result := make([]byte, 0, len)

	result = append(result, no, byte(code))
	result = append(result, data...)

	return result
}

func (c *CanConverter) SendFrame(data []byte) {
	// c.Tx <- combineCanData(c.No, c.Code, data)

	go sendData(combineCanData(c.No, c.Code, data), c.Tx)

}

func CanRegist(id []byte, no byte, tx chan<- []byte) {

	data := make([]byte, 0, 8)
	data = append(data, no)
	data = append(data, id...)
	go sendData(combineCanData(0, 0, data), tx)

}

func (c *CanConverter) HeartRequest() {

	// c.Tx <- combineCanData(c.No, 4, []byte{0x11, 0x22})
	go sendData(combineCanData(c.No, 4, []byte{0x11, 0x22}), c.Tx)

}

func (c *CanConverter) GetSN() string {
	return c.SN
}

func combineLoraData(cmd byte, id []byte, data []byte) []byte {
	len := 8 + len(data)

	result := make([]byte, 0, len)
	result = append(result, 0xA5)
	result = append(result, byte(len))
	result = append(result, cmd)
	result = append(result, id...)
	result = append(result, data...)

	sum := byte(0)
	for _, v := range result {
		sum += v
	}
	result = append(result, sum)

	return result
}

func (c *LoraConverter) Setting485(speed uint32, event byte, data byte, stop byte) {
	d := []byte{byte(speed), byte(speed >> 8), byte(speed >> 16), byte(speed >> 24), data, event, stop}
	// c.Tx <- combineLoraData(0x06, c.Id, d)

	go sendData(combineLoraData(0x06, c.Id, d), c.Tx)

}

func (c *LoraConverter) SendFrame(data []byte) {
	// c.Tx <- combineLoraData(c.Cmd, c.Id, data)
	d := combineLoraData(c.Cmd, c.Id, data)

	go sendData(d, c.Tx)
}

func LoraRegist(id []byte, tx chan<- []byte) {

	// {chanel, password}
	data := []byte{0x17, 0x00, 0x00}

	// tx <- combineLoraData(0x01, id, data)

	d := combineLoraData(0x01, id, data)
	go sendData(d, tx)

}

func (c *LoraConverter) HeartRequest() {

	var cmd byte

	switch c.LoraType {

	// io
	case 4:
		cmd = 0x14

	// 485
	case 5:
		cmd = 0x05
	}

	// c.Tx <- combineLoraData(cmd, c.Id, []byte{})
	d := combineLoraData(cmd, c.Id, []byte{})
	go sendData(d, c.Tx)

}

func (c *LoraConverter) GetSN() string {
	return c.SN
}

// func combineSerialData(addr byte, data []byte) []byte {
// 	len := 1 + len(data)

// 	result := make([]byte, 0, len)

// 	result = append(result, addr)
// 	result = append(result, data...)

// 	return result

// }

func (c *SerialConverter) SendFrame(data []byte) {

	// c.Tx <- combineSerialData(c.Addr, data)
	// d := combineSerialData(c.Addr, data)

	go sendData(data, c.Tx)
}

func (c *SerialConverter) GetSN() string {
	return c.SN
}

func (c *SerialConverter) GetAddr() uint8 {
	return c.Addr
}

func (c *SerialConverter) HeartRequest() {
}
