//+build windows

package aida

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	iconv "github.com/djimenez/iconv-go"
	"github.com/gorpher/idpc-plugins/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	AIDA_CMD_PARAMS = []string{"/SILENT", "/LANGen", "/XML", "/NOICONS", "/NOLICENSE", "/R"}
)

type AIDA_CMD string

const (
	AIDA_NAME                = "aida64.exe"
	AIDA_CMD_ALL    AIDA_CMD = "/ALL"
	AIDA_CMD_SUM    AIDA_CMD = "/SUM"
	AIDA_CMD_HW     AIDA_CMD = "/HW" //hardware
	AIDA_CMD_SW     AIDA_CMD = "/SW" //software
	AIDA_CMD_BENCH  AIDA_CMD = "/BENCH"
	AIDA_CMD_AUDIT  AIDA_CMD = "/AUDIT"
	AIDA_CMD_CUSTOM AIDA_CMD = "/CUSTOM"
)

type AidaWrapper struct {
	execPath         string
	customConfigPath string
}

func NewAidaWrapper(execPath, customConfigPath string) (*AidaWrapper, error) {
	aidaPath, err := finAidaExeFile(execPath)
	if err != nil {
		return nil, err
	}
	return &AidaWrapper{execPath: aidaPath, customConfigPath: customConfigPath}, nil
}

// CallCMD 同步阻塞调用命令行工具
// name 程序名称，建议使用绝对路径
// params 程序启动参数
func CallCMD(name string, params []string) error {
	command := exec.Command(name, params...)
	exeble, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exeble)
	if filepath.IsAbs(name) {
		dir = filepath.Dir(name)
	} else {
		if fileAbs, err := filepath.Abs(name); err == nil {
			dir = filepath.Dir(fileAbs)
		}
	}
	command.Dir = dir
	return command.Run()
}

// finAidaExeFile 获取aida可执行文件的绝对路径
func finAidaExeFile(aidaPath string) (string, error) {
	return utils.ScanFile(aidaPath, AIDA_NAME, "AIDA_PATH")
}

// findAidaConfigFile 获取aida report 配置文件路径，如果没有返回空字符串
func findAidaConfigFile(aidaConfigPath string) string {
	var filePath string
	if f := aidaConfigPath; utils.FileExist(f) {
		filePath = f
	}
	if filePath == "" {
		f := os.Getenv("AIDA_CUSTOM_PATH")
		if utils.FileExist(f) {
			filePath = f
		}
	}
	if filePath == "" {
		executable, _ := os.Executable()
		f := filepath.Join(filepath.Dir(executable), "custom.rpf")
		if utils.FileExist(f) {
			filePath = f
		}
	}
	return filePath
}

// CallAIDA 调用AIDA命令行工具
func (a *AidaWrapper) CallAIDA(acmd AIDA_CMD) (map[string]interface{}, error) {
	var (
		reportPath = getReportName(acmd)
		err        error
		reportFile *os.File
	)
	params := append(AIDA_CMD_PARAMS, reportPath, string(acmd))
	if acmd == AIDA_CMD_CUSTOM {
		configFile := findAidaConfigFile(a.customConfigPath)
		if configFile == "" {
			return nil, errors.New("自定义的报告配置文件不存在")
		}
		params = append(params, a.customConfigPath)
	}
	log.Debug().Msg(strings.Join(params, " "))
	if err = CallCMD(a.execPath, params); err != nil {
		if strings.Contains(err.Error(), "The requested operation requires elevation") {
			return nil, errors.New("执行该程序需要管理员权限")
		}
		return nil, err
	}
	reportFile, err = os.OpenFile(reportPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer reportFile.Close() // nolint
	reportBytes, err := io.ReadAll(reportFile)
	if err != nil {
		return nil, err
	}
	report, err := parseXML(reportBytes)
	if err != nil {
		return nil, err
	}
	return parseReport(acmd, report)
}

func getReportName(acmd AIDA_CMD) string {
	executable, _ := os.Executable()
	return filepath.Join(filepath.Dir(executable), fmt.Sprintf("report_%s_%d.xml", strings.ToLower(strings.TrimLeft(string(acmd), "/")), time.Now().UTC().Unix()))
}

func parseXML(data []byte) (report Report, err error) {
	converter, _ := iconv.NewConverter("gb2312", "utf-8")
	output, _ := converter.ConvertString(string(data))
	//err = xml.Unmarshal(bytes.Replace(data, []byte("encoding=\"iso-8859-1\""), []byte("encoding=\"UTF-8\""), 1), &report)
	err = xml.Unmarshal(bytes.Replace([]byte(output), []byte("encoding=\"iso-8859-1\""), []byte("encoding=\"UTF-8\""), 1), &report)
	return report, err
}

func parseReport(acmd AIDA_CMD, report Report) (map[string]interface{}, error) {
	var res = map[string]interface{}{}
	if acmd == AIDA_CMD_SUM {
		summary := Summary{
			Computer:    []KV{},
			Motherboard: []KV{},
			Display:     []KV{},
			Multimedia:  []KV{},
			Storage:     []KV{},
			Partitions:  []KV{},
			Input:       []KV{},
			Network:     []KV{},
			Peripherals: []KV{},
			DMI:         []KV{},
		}
		if len(report.Page) == 2 {
			page2 := report.Page[1]
			for k := range page2.Group {
				sum := page2.Group[k]
				for i := range sum.Item {
					if sum.Title == "Computer" {
						summary.Computer = append(summary.Computer, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Motherboard" {
						summary.Motherboard = append(summary.Motherboard, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Display" {
						summary.Display = append(summary.Display, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Multimedia" {
						summary.Multimedia = append(summary.Multimedia, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Storage" {
						summary.Storage = append(summary.Storage, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Partitions" {
						summary.Partitions = append(summary.Partitions, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Input" {
						summary.Input = append(summary.Input, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Network" {
						summary.Network = append(summary.Network, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "Peripherals" {
						summary.Peripherals = append(summary.Peripherals, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
					if sum.Title == "DMI" {
						summary.DMI = append(summary.DMI, KV{Title: sum.Item[i].Title, Value: sum.Item[i].Value})
					}
				}
			}
		}
		res["compute"] = summary.Computer
		res["motherboard"] = summary.Motherboard
		res["display"] = summary.Display
		res["multimedia"] = summary.Multimedia
		res["storage"] = summary.Storage
		res["partitions"] = summary.Partitions
		res["input"] = summary.Input
		res["network"] = summary.Network
		res["peripherals"] = summary.Peripherals
		res["dmi"] = summary.DMI
		return res, nil
	}
	if acmd == AIDA_CMD_HW {
		log.Debug().Interface("report", report).Send()
	}
	return res, nil
}

func toUtf8(iso8859_1_buf []byte) string {
	buf := make([]rune, len(iso8859_1_buf))
	for i, b := range iso8859_1_buf {
		buf[i] = rune(b)
	}
	return string(buf)
}

func UTF82GB2312(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.HZGB2312.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}
