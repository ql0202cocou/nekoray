package main

import (
	"github.com/matsuridayo/libneko/neko_log"
	sBlog "github.com/sagernet/sing-box/log"
)

// nekoPlatformWriter bridges sing-box PlatformWriter to neko_log
type nekoPlatformWriter struct{}

func (w *nekoPlatformWriter) WriteMessage(level sBlog.Level, message string) {
	neko_log.LogWriter.Write([]byte(message + "\n"))
}
