package storage

import (
	"bambuscribe/desktop/internal/calculation"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/printer"
)

type Preferences struct {
	Calculations []calculation.Record   `json:"calculations,omitempty"`
	Erasures     []engine.PageErasure   `json:"erasures,omitempty"`
	Theme        string                 `json:"theme"`
	Text         string                 `json:"text"`
	Layout       engine.LayoutSettings  `json:"layout"`
	Writing      engine.WritingSettings `json:"writing"`
	Language     string                 `json:"language,omitempty"`
	Images       []engine.ImageAsset    `json:"images,omitempty"`
	Tables       []engine.TableAsset    `json:"tables,omitempty"`
}

func Defaults() Preferences {
	return Preferences{
		Theme:    "light",
		Text:     "把文字，写在纸上。\n\n清晨的风穿过窗边，阳光落在书桌上。慢慢写下今天的想法，让每一个字都有自己的温度。\n\n有些事情不必着急，一笔一画，也是一种认真生活的方式。\n\n你好，新的开始。",
		Layout:   engine.DefaultLayout(),
		Writing:  engine.DefaultWriting(),
		Language: "system",
		Images:   []engine.ImageAsset{},
	}
}

type Data struct {
	Version          int              `json:"version"`
	ProjectPath      string           `json:"projectPath,omitempty"`
	Preferences      Preferences      `json:"preferences"`
	Printer          printer.Config   `json:"printer"`
	Devices          []printer.Device `json:"devices"`
	SelectedDeviceID string           `json:"selectedDeviceId"`
}
