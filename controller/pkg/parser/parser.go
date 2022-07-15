package parser

import (
	pkgSw "controller/pkg/p4switch"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ActionDescr struct {
	ActionName   string
	ActionId     int
	ActionParams []pkgSw.FieldDescriber
}

const (
	pathJsonInfo = "../../../p4/JsonOfP4info/"
	extJsonInfo  = ".p4.p4info.json"
)

func main() {
	fmt.Println("\n--- simple ---\n\n", parseP4Info("simple"))
	fmt.Println("\n--- asymmetric ---\n\n", parseP4Info("asymmetric"))
	fmt.Println("\n--- simple1 ---\n\n", parseP4Info("simple1"))
}

func findIfKnownPattern(name string, bitwidth int) *string {
	return nil
}

func parseToBytesKeysOf(rule pkgSw.Rule) [][]byte {
	return nil
}

func parseToBytesActionParamsOf(rule pkgSw.Rule) [][]byte {
	return nil
}

func parseP4Info(p4Program string) string { // return json of []pkgSw.RuleDescriber

	filename := pathJsonInfo + p4Program + extJsonInfo

	jsonFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		// FIXME return "" ?
		return ""
	}

	fmt.Print("\n[DEBUG] Successfully Opened ", filename, "\n")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}

	json.Unmarshal([]byte(byteValue), &result)

	// Extract actions informations

	var actions_descr []ActionDescr

	var describers []pkgSw.RuleDescriber

	for index_actions := range result["actions"].([]interface{}) {

		action := (result["actions"].([]interface{})[index_actions]).(map[string]interface{})

		preamble := action["preamble"].(map[string]interface{})
		action_name := preamble["name"].(string)
		action_id := int(preamble["id"].(float64))

		action_parameters := []pkgSw.FieldDescriber{}

		if action["params"] != nil {
			for index_parameters := range action["params"].([]interface{}) {
				parameter := action["params"].([]interface{})[index_parameters].(map[string]interface{})

				action_parameters = append(action_parameters, pkgSw.FieldDescriber{
					Name:     parameter["name"].(string),
					Bitwidth: int(parameter["bitwidth"].(float64)),
					// Pattern: "",
				})
			}
		}

		actions_descr = append(actions_descr, ActionDescr{
			ActionName:   action_name,
			ActionId:     action_id,
			ActionParams: action_parameters,
		})

	}
	// Extract tables informations

	for i := range result["tables"].([]interface{}) {

		table := result["tables"].([]interface{})[i].(map[string]interface{})

		preamble := table["preamble"].(map[string]interface{})
		table_name := preamble["name"].(string)
		table_id := int(preamble["id"].(float64))

		var talbe_keys []pkgSw.FieldDescriber

		for index_keys := range table["matchFields"].([]interface{}) {

			key := table["matchFields"].([]interface{})[index_keys].(map[string]interface{})

			talbe_keys = append(talbe_keys, pkgSw.FieldDescriber{
				Name:     key["name"].(string),
				Bitwidth: int(key["bitwidth"].(float64)),
				// Pattern: "",
			})
		}

		var actions_ids []int
		for _, action_refs := range table["actionRefs"].([]interface{}) {
			actions_ids = append(actions_ids, int(action_refs.(map[string]interface{})["id"].(float64)))
		}

		// find action contained in actual table and then create a new describer
		// if a table has no action or an action doesn't refer to a table, them won't be added to result

		for _, ac := range actions_descr {

			if integer_contains(actions_ids, ac.ActionId) {

				describers = append(describers, pkgSw.RuleDescriber{
					TableName:    table_name,
					TableId:      table_id,
					Keys:         talbe_keys,
					ActionName:   ac.ActionName,
					ActionId:     ac.ActionId,
					ActionParams: ac.ActionParams,
				})

			}
		}
	}
	resInByte, err := json.Marshal(describers)
	if err != nil {
		return ""
	}
	res := string(resInByte)

	return res
}

func getDescriberFor(p4Program string, rule pkgSw.Rule) *pkgSw.RuleDescriber {
	return nil
}

func integer_contains(array []int, value int) bool {
	for _, el := range array {
		if el == value {
			return true
		}
	}
	return false
}
