//+build windows

package aida

import (
	"flag"
	"fmt"
	"github.com/gorpher/go-idpc-plugin"
	"runtime"
)

type AidaPlugin struct {
	plugin.MetadataPlugin
	Key     string
	wrapper *AidaWrapper
}

func (r AidaPlugin) Metadata() (map[string]interface{}, error) {
	return r.wrapper.CallAIDA(AIDA_CMD_SUM)
}

var (
	Revision  = "untracked"
	Version   = "0.0.0"
	GOARCH    = runtime.GOARCH
	GOOS      = runtime.GOOS
	GOVersion = runtime.Version()
)

func (r AidaPlugin) Meta() plugin.Meta {
	if r.Key == "" {
		r.Key = "aida"
	}
	version, _ := plugin.ParseVersion(Version)
	return plugin.Meta{
		Key:       r.Key,
		Type:      plugin.TypeMetadata,
		Version:   version,
		Revision:  Revision,
		GOARCH:    GOARCH,
		GOOS:      GOOS,
		GOVersion: GOVersion,
	}
}

// Do the plugin
func Do() {
	optTempFile := flag.String("tempFile", "", "Temp file name")
	exeFile := flag.String("exePath", "", "exe file path")
	cFile := flag.String("configPath", "", "config file path")

	v := flag.Bool("v", false, "version")
	flag.Parse()

	wrapper, err := NewAidaWrapper(*exeFile, *cFile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	aida := AidaPlugin{
		wrapper: wrapper,
	}

	helper := plugin.NewIdpcPlugin(aida)
	if *v {
		fmt.Println(helper.Version())
		return
	}
	if *optTempFile != "" {
		helper.TempFile = *optTempFile
	}
	helper.Run()
}
