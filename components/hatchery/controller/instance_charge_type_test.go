package controller

import (
	"encoding/json"
	"testing"
)

func TestApplyInstanceChargeTypeToCVMTemplate_PreservesPrepaidConfigWhenPostpaid(t *testing.T) {
	input := `{
		"InstanceChargeType":"PREPAID",
		"InstanceType":"S5.SMALL1",
		"InstanceChargePrepaid":{"Period":3,"RenewFlag":"NOTIFY_AND_MANUAL_RENEW"}
	}`

	got, err := applyInstanceChargeTypeToCVMTemplate(input, "POSTPAID_BY_HOUR")
	if err != nil {
		t.Fatalf("applyInstanceChargeTypeToCVMTemplate returned error: %v", err)
	}

	var tpl map[string]interface{}
	if err := json.Unmarshal([]byte(got), &tpl); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if tpl["InstanceChargeType"] != cvmChargeTypePostpaidByHour {
		t.Fatalf("InstanceChargeType = %v, want %s", tpl["InstanceChargeType"], cvmChargeTypePostpaidByHour)
	}
	prepaid, ok := tpl["InstanceChargePrepaid"].(map[string]interface{})
	if !ok {
		t.Fatalf("InstanceChargePrepaid missing or wrong type: %#v", tpl["InstanceChargePrepaid"])
	}
	if prepaid["Period"] != float64(3) || prepaid["RenewFlag"] != "NOTIFY_AND_MANUAL_RENEW" {
		t.Fatalf("InstanceChargePrepaid changed: %#v", prepaid)
	}
}

func TestApplyInstanceChargeTypeToCVMTemplate_DoesNotAddPrepaidConfigWhenPrepaid(t *testing.T) {
	input := `{"InstanceChargeType":"POSTPAID_BY_HOUR","InstanceType":"S5.SMALL1"}`

	got, err := applyInstanceChargeTypeToCVMTemplate(input, "PREPAID")
	if err != nil {
		t.Fatalf("applyInstanceChargeTypeToCVMTemplate returned error: %v", err)
	}

	var tpl map[string]interface{}
	if err := json.Unmarshal([]byte(got), &tpl); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if tpl["InstanceChargeType"] != cvmChargeTypePrepaid {
		t.Fatalf("InstanceChargeType = %v, want %s", tpl["InstanceChargeType"], cvmChargeTypePrepaid)
	}
	if _, ok := tpl["InstanceChargePrepaid"]; ok {
		t.Fatalf("InstanceChargePrepaid should not be added: %#v", tpl["InstanceChargePrepaid"])
	}
}

func TestApplyInstanceChargeTypeToCVMTemplate_NormalizesAndRejectsInvalid(t *testing.T) {
	got, err := applyInstanceChargeTypeToCVMTemplate(`{"InstanceType":"S5.SMALL1"}`, " postpaid_by_hour ")
	if err != nil {
		t.Fatalf("lowercase charge type should be accepted: %v", err)
	}
	var tpl map[string]interface{}
	if err := json.Unmarshal([]byte(got), &tpl); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if tpl["InstanceChargeType"] != cvmChargeTypePostpaidByHour {
		t.Fatalf("InstanceChargeType = %v, want %s", tpl["InstanceChargeType"], cvmChargeTypePostpaidByHour)
	}

	if _, err := applyInstanceChargeTypeToCVMTemplate(`{"InstanceType":"S5.SMALL1"}`, "SPOTPAID"); err == nil {
		t.Fatal("expected invalid charge type error")
	}
}

func TestInstanceChargeTypeOrDefault(t *testing.T) {
	if got := instanceChargeTypeOrDefault(""); got != cvmChargeTypePrepaid {
		t.Fatalf("empty charge type = %s, want %s", got, cvmChargeTypePrepaid)
	}
	if got := instanceChargeTypeOrDefault(" SPOTPAID "); got != "SPOTPAID" {
		t.Fatalf("non-empty charge type = %s, want SPOTPAID", got)
	}
}

func TestCVMTemplateMapNull(t *testing.T) {
	got, err := cvmTemplateMap("null")
	if err != nil {
		t.Fatalf("cvmTemplateMap(null) returned error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("cvmTemplateMap(null) = %#v, want empty map", got)
	}
}

func TestInstanceChargeTypeFromCVMTemplate(t *testing.T) {
	tests := []struct {
		name string
		tpl  string
		want string
	}{
		{name: "explicit prepaid", tpl: `{"InstanceChargeType":"PREPAID"}`, want: cvmChargeTypePrepaid},
		{name: "normalized", tpl: `{"InstanceChargeType":" postpaid_by_hour "}`, want: cvmChargeTypePostpaidByHour},
		{name: "missing uses Tencent default", tpl: `{"InstanceType":"S5.SMALL1"}`, want: cvmChargeTypePostpaidByHour},
		{name: "empty template uses project default", tpl: ``, want: cvmChargeTypePrepaid},
		{name: "invalid template uses safe fallback", tpl: `not-json`, want: cvmChargeTypePrepaid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := instanceChargeTypeFromCVMTemplate(tt.tpl); got != tt.want {
				t.Fatalf("instanceChargeTypeFromCVMTemplate() = %s, want %s", got, tt.want)
			}
		})
	}
}
