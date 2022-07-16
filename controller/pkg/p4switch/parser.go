package p4switch

import (
	"controller/pkg/client"
	"controller/pkg/util/conversion"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	pattern_ipv4_addr = "ipv4_address"
	pattern_mac_addr  = "mac_address"
	pattern_port      = "port"
)

type ActionDescr struct {
	ActionName   string
	ActionId     int
	ActionParams []FieldDescriber
}
type ParserMatchInterface interface {
	parse(keys []string, describers []FieldDescriber) []client.MatchInterface
}

type ExactMatchParser struct {
	Type string
}

func (p *ExactMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {
	// TODO
	return nil
}

type TernaryMatchParser struct {
	Type string
}

func (p *TernaryMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {
	// TODO
	return nil
}

type LpmMatchParser struct {
	Type string
}

func (p *LpmMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {
	// TODO
	return nil
}

func getParserForMatchInterface(parserType string) ParserMatchInterface {

	switch strings.ToLower(parserType) {
	case "exact":
		return ParserMatchInterface(&ExactMatchParser{
			Type: parserType,
		})
	case "lpm":
		return ParserMatchInterface(&LpmMatchParser{
			Type: parserType,
		})
	case "ternary":
		return ParserMatchInterface(&TernaryMatchParser{
			Type: parserType,
		})
	default:
		return nil
	}
}

type ParserActionParams interface {
	parse(params []string, describers []FieldDescriber) [][]byte
}

type DefaultParserActionParams struct{}

func (p *DefaultParserActionParams) parse(params []string, describers []FieldDescriber) [][]byte {

	actionByte := make([][]byte, len(params))

	for idx, par := range params {

		switch describers[idx].Pattern {

		case pattern_mac_addr:
			actionByte[idx], _ = conversion.MacToBinary(par)

		case pattern_ipv4_addr:
			actionByte[idx], _ = conversion.IpToBinary(par)

		case pattern_port:
			{
				num, _ := strconv.ParseInt(par, 10, 64)
				actionByte[idx], _ = conversion.UInt64ToBinaryCompressed(uint64(num))
			}
		}

	}
	return actionByte
}

func getParserForActionParams(parserType string) ParserActionParams {
	return ParserActionParams(&DefaultParserActionParams{})
}

const (
	pathJsonInfo = "../../../p4/JsonOfP4info/"
	extJsonInfo  = ".p4.p4info.json"
)

func parseP4Info(p4Program string) *string { // return json of []RuleDescriber

	filename := pathJsonInfo + p4Program + extJsonInfo

	jsonFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	fmt.Print("\n[DEBUG] Successfully Opened ", filename, "\n")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}

	json.Unmarshal([]byte(byteValue), &result)

	// Extract actions informations

	var actions_descr []ActionDescr

	var describers []RuleDescriber

	for index_actions := range result["actions"].([]interface{}) {

		action := (result["actions"].([]interface{})[index_actions]).(map[string]interface{})

		preamble := action["preamble"].(map[string]interface{})
		action_name := preamble["name"].(string)
		action_id := int(preamble["id"].(float64))

		action_parameters := []FieldDescriber{}

		if action["params"] != nil {
			for index_parameters := range action["params"].([]interface{}) {
				parameter := action["params"].([]interface{})[index_parameters].(map[string]interface{})

				action_parameters = append(action_parameters, FieldDescriber{
					Name:     parameter["name"].(string),
					Bitwidth: int(parameter["bitwidth"].(float64)),
					Pattern:  findIfKnownPattern(parameter["name"].(string), int(parameter["bitwidth"].(float64))),
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

		var talbe_keys []FieldDescriber

		var matchType string

		for index_keys := range table["matchFields"].([]interface{}) {

			key := table["matchFields"].([]interface{})[index_keys].(map[string]interface{})

			matchType = key["matchType"].(string)

			talbe_keys = append(talbe_keys, FieldDescriber{
				Name:     key["name"].(string),
				Bitwidth: int(key["bitwidth"].(float64)),
				// TODO Add mask if ternary match
				Pattern: findIfKnownPattern(key["name"].(string), int(key["bitwidth"].(float64))),
			})
		}

		var actions_ids []int
		for _, action_refs := range table["actionRefs"].([]interface{}) {
			actions_ids = append(actions_ids, int(action_refs.(map[string]interface{})["id"].(float64)))
		}

		// find action contained in actual table and then create a new describer
		// if a table has no action or an action doesn't refer to a table, them won't be added to result

		for _, ac := range actions_descr {

			if contains_int(actions_ids, ac.ActionId) {

				describers = append(describers, RuleDescriber{
					TableName:    table_name,
					TableId:      table_id,
					MatchType:    matchType,
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
		return nil
	}
	res := string(resInByte)

	return &res
}

func getDescriberFor(p4Program string, rule Rule) *RuleDescriber {

	res := *parseP4Info(p4Program)

	var describers []RuleDescriber

	json.Unmarshal([]byte(res), &describers)

	for _, descr := range describers {
		if rule.Action == descr.ActionName && rule.Table == descr.TableName {
			return &descr
		}
	}

	return nil
}

func findIfKnownPattern(name string, bitwidth int) string {
	if strings.Contains(strings.ToLower(name), "port") {
		return pattern_port
	}
	if strings.Contains(strings.ToLower(name), "addr") {
		switch bitwidth {
		case 32:
			return pattern_ipv4_addr // ipv4 address pattern
		case 48:
			return pattern_mac_addr // mac address pattern
		}
	}
	return ""
}

func parseLPMKey(value string, describer FieldDescriber) []byte {
	return nil
}

func parseEXACTKey(value string, describer FieldDescriber) []byte {
	return nil
}

func parseTERNARYKey(value string, describer FieldDescriber) []byte {
	return nil
}

func parseKeysToMatchInterfaceFrom(rule Rule) []client.MatchInterface {
	keysBytes := make([][]byte, len(rule.Key))

	for idx, key := range rule.Key {

		switch rule.Describer.MatchType {
		case "ipv4_address":
			keysBytes[idx], _ = conversion.IpToBinary(key)
		case "mac_address":
			keysBytes[idx], _ = conversion.MacToBinary(key)
		case "port":
			{
				intValue, _ := strconv.Atoi(key)
				keysBytes[idx], _ = conversion.UInt64ToBinaryCompressed(uint64(intValue))
			}
		default: //TODO: do none (?)
		}
	}
	return nil

}

func parseActionParamsToBytesFrom(rule Rule) [][]byte {
	return nil
}

func contains_int(array []int, value int) bool {
	for _, el := range array {
		if el == value {
			return true
		}
	}
	return false
}
