package aida

import "encoding/xml"

type ReportXML struct {
	Encoding string `xml:"encoding,attr"`
	Version  string `xml:"version,attr"`
	//Report Report `xml:",innerxml"`
}
type Report struct {
	XMLName xml.Name `xml:"Report"`
	Lang    string
	Page    []Page
}
type Page struct {
	XMLName xml.Name `xml:"Page"`

	Title string
	Icon  int
	Item  []Item

	MenuTitle string
	MenuIcon  int
	Group     []Group

	Device []Device
}

type Device struct {
	XMLName xml.Name `xml:"Device"`

	Title string
	Icon  int
	Item  []Item
	Group []Group
}
type Item struct {
	XMLName xml.Name `xml:"Item"`

	Title string
	Icon  int
	ID    int
	Value string
}
type Group struct {
	XMLName xml.Name `xml:"Group"`

	Title string
	Icon  int
	Item  []Item
}

type KV struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type Summary struct {
	Computer    []KV // 计算机
	Motherboard []KV // 主板
	Display     []KV // 显卡
	Multimedia  []KV // 声卡
	Storage     []KV // 存储
	Partitions  []KV // 分区
	Input       []KV // 输入设备
	Network     []KV // 网卡
	Peripherals []KV // 打印机
	DMI         []KV // DMI
}
