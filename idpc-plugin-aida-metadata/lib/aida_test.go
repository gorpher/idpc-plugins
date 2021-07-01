//+build windows

package aida

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

const AIDAPath = "D:/space/idp-client-packaging/aida64business/aida64.exe"
const AIDACustomPath = "D:/space/idp-client-packaging/custom.rpf"
const TEST_PATH = "D:/space/idpc-plugins/idpc-plugin-aida-metadata/test/xml"

func TestScanAIDA(t *testing.T) {
	if _, err := finAidaExeFile(AIDAPath); err != nil {
		t.Error(err)
		return
	}
}

func TestCallAIDA(t *testing.T) {
	var data map[string]interface{}
	var v []byte
	var err error
	aidaWrapper, err := NewAidaWrapper(AIDAPath, AIDACustomPath)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_SUM); err != nil {
		if strings.Contains(err.Error(), "The requested operation requires elevation") {
			t.Error("需要管理员权限测试")
		}
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_SUM=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_HW); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_HW=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_SW); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_SW=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_AUDIT); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_AUDIT=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))

}
func TestCallAIDA2(t *testing.T) {
	var data map[string]interface{}
	var v []byte
	var err error
	aidaWrapper, err := NewAidaWrapper(AIDAPath, AIDACustomPath)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_BENCH); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_BENCH=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
}

func TestCallAIDA3(t *testing.T) {
	var data map[string]interface{}
	var v []byte
	var err error
	aidaWrapper, err := NewAidaWrapper(AIDAPath, AIDACustomPath)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_ALL); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_ALL=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
}

func TestTestCallAIDA_CUSTOM(t *testing.T) {
	var data map[string]interface{}
	var v []byte
	var err error
	aidaWrapper, err := NewAidaWrapper(AIDAPath, AIDACustomPath)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err = aidaWrapper.CallAIDA(AIDA_CMD_CUSTOM); err != nil {
		t.Error(err)
		return
	}
	t.Log("==========================================AIDA_CMD_CUSTOM=========================================")
	v, _ = json.Marshal(data)
	t.Log(string(v))
}

func TestParseReportXML(t *testing.T) {
	var testSets = map[AIDA_CMD]string{
		AIDA_CMD_SUM:   filepath.Join(TEST_PATH, "sum.xml"),
		AIDA_CMD_HW:    filepath.Join(TEST_PATH, "hw.xml"),
		AIDA_CMD_SW:    filepath.Join(TEST_PATH, "sw.xml"),
		AIDA_CMD_AUDIT: filepath.Join(TEST_PATH, "audit.xml"),
	}
	for acmd := range testSets {
		testParsePort(t, acmd, testSets[acmd])
	}
}

func testParsePort(t *testing.T, acmd AIDA_CMD, reportPath string) {
	var resData map[string]interface{}
	var v []byte
	var err error
	testData, err := ioutil.ReadFile(reportPath)
	if err != nil {
		t.Error(err)
		return
	}
	report, err := parseXML(testData)
	if err != nil {
		t.Error(err)
		return
	}
	if resData, err = parseReport(acmd, report); err != nil {
		t.Error("parse"+acmd, err)
		return
	}
	v, _ = json.Marshal(resData)
	t.Log(string(v))
	t.Log("parse"+acmd, string(v))
}
