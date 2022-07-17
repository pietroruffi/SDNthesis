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

	pathJsonInfo = "../../../p4/JsonOfP4info/"
	extJsonInfo  = ".p4.p4info.json"
)

// ActionDescr used in parseP4Info() to keep all fields of an action together, and not in different arrays where elements related have same index
type ActionDescr struct {
	ActionName   string
	ActionId     int
	ActionParams []FieldDescriber
}

// Define general parser for MatchInterfaces
type ParserMatchInterface interface {
	parse(keys []string, describers []FieldDescriber) []client.MatchInterface
}

// Specific parser for MatchInterfaces with matchType: "exact"
type ExactMatchParser struct {
}

func (p *ExactMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {

	var result []client.MatchInterface
	var field []byte

	for idx, key := range keys {
		switch describers[idx].Pattern {
		// pattern is added in parseP4Info(), defines if the key satisfies a known pattern and had to be parsed in a specific way
		case pattern_mac_addr:
			{
				field, _ = conversion.MacToBinary(key)
			}
		case pattern_ipv4_addr:
			{
				field, _ = conversion.IpToBinary(key)
			}
		case pattern_port:
			{
				num, _ := strconv.ParseInt(key, 10, 64)
				field, _ = conversion.UInt64ToBinaryCompressed(uint64(num))
			}
		}
		// add to result the key trasformed into []byte
		result = append(result, client.MatchInterface(&client.ExactMatch{
			Value: field,
		}))
	}
	return result
}

// Specific parser for MatchInterfaces with matchType: "lpm"
type LpmMatchParser struct {
}

func (p *LpmMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {

	var result []client.MatchInterface
	var field []byte
	var lpm int64

	for idx, key := range keys {
		switch describers[idx].Pattern {
		case pattern_ipv4_addr:
			{
				// TODO write better
				values := strings.Split(key, "/")
				if len(values) != 2 {
					//log.Errorf("Error parsing match %s -> %s", matchType, key)
					return nil
				}
				field, _ = conversion.IpToBinary(values[0])
				lpm, _ = strconv.ParseInt(values[1], 10, 64)
				/*if err != nil {
					//log.Errorf("Error parsing lpm %d", lpm)
				}*/
			}
		case pattern_mac_addr:
			// TODO

		case pattern_port:
			// TODO

		default:
			// TODO
		}
		result = append(result, client.MatchInterface(&client.LpmMatch{
			Value: field,
			PLen:  int32(lpm),
		}))
	}
	return result
}

// Specific parser for MatchInterfaces with matchType: "ternary"
type TernaryMatchParser struct {
}

func (p *TernaryMatchParser) parse(keys []string, describers []FieldDescriber) []client.MatchInterface {
	// CHANGE search how to parse ternary
	var result []client.MatchInterface
	var field []byte
	var lpm int64
	for idx, key := range keys {
		switch describers[idx].Pattern {
		case pattern_ipv4_addr:
			{
				values := strings.Split(key, "/")
				if len(values) != 2 {
					//log.Errorf("Error parsing match %s -> %s", matchType, key)
					return nil
				}
				field, _ = conversion.IpToBinary(values[0])
				lpm, _ = strconv.ParseInt(values[1], 10, 64)
				/*if err != nil {
					//log.Errorf("Error parsing lpm %d", lpm)
				}*/
			}
		case pattern_mac_addr:
			// TODO

		case pattern_port:
			// TODO

		default:
			// TODO
		}
		result = append(result, client.MatchInterface(&client.LpmMatch{
			Value: field,
			PLen:  int32(lpm),
		}))
	}
	return result
}

// GATTINO

// A kind of "ParserFactory", returns the parser for the specified matchType (exact | lpm | ternary)

func getParserForMatchInterface(parserType string) ParserMatchInterface {

	switch strings.ToLower(parserType) {
	case "exact":
		return ParserMatchInterface(&ExactMatchParser{})
	case "lpm":
		return ParserMatchInterface(&LpmMatchParser{})
	case "ternary":
		return ParserMatchInterface(&TernaryMatchParser{})
	default:
		return nil
	}
}

// Define general parser for ActionParameters
type ParserActionParams interface {
	parse(params []string, describers []FieldDescriber) [][]byte
}

// There's no need to define more than one parser, because ActionParameters are not influenced by matchType
// but to keep everything more general (and for future extensions) is defined a general structure and a default parser

type DefaultParserActionParams struct{}

func (p *DefaultParserActionParams) parse(params []string, describers []FieldDescriber) [][]byte {

	actionByte := make([][]byte, len(params))
	var field []byte

	for idx, par := range params {
		switch describers[idx].Pattern {
		case pattern_mac_addr:
			{
				field, _ = conversion.MacToBinary(par)
			}
		case pattern_ipv4_addr:
			{
				field, _ = conversion.IpToBinary(par)
			}
		case pattern_port:
			{
				num, _ := strconv.ParseInt(par, 10, 64)
				field, _ = conversion.UInt64ToBinaryCompressed(uint64(num))
			}
		}
		actionByte[idx] = field

	}
	return actionByte
}

// As said before there is only one parser for ActionParameters, so we return that one regardless of parserType

func getParserForActionParams(parserType string) ParserActionParams {
	return ParserActionParams(&DefaultParserActionParams{})
}

func parseP4Info(p4Program string) *string { // return json of []RuleDescriber

	filename := pathJsonInfo + p4Program + extJsonInfo

	jsonFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var jsonResult map[string]interface{}

	json.Unmarshal([]byte(byteValue), &jsonResult)

	// Define result variable

	var result []RuleDescriber

	// Extract actions informations

	var actions_descr []ActionDescr

	for index_actions := range jsonResult["actions"].([]interface{}) {

		action := (jsonResult["actions"].([]interface{})[index_actions]).(map[string]interface{})

		preamble := action["preamble"].(map[string]interface{})
		action_name := preamble["name"].(string)
		action_id := int(preamble["id"].(float64))

		action_parameters := []FieldDescriber{}

		// If there are some ActionParams, extract them

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

	for i := range jsonResult["tables"].([]interface{}) {

		table := jsonResult["tables"].([]interface{})[i].(map[string]interface{})

		preamble := table["preamble"].(map[string]interface{})
		table_name := preamble["name"].(string)
		table_id := int(preamble["id"].(float64))

		var talbe_keys []FieldDescriber

		var matchType string

		// Extract keys

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

		// Extract IDs of actions the actual table offers

		var actions_ids []int
		for _, action_refs := range table["actionRefs"].([]interface{}) {
			actions_ids = append(actions_ids, int(action_refs.(map[string]interface{})["id"].(float64)))
		}

		// find actions contained in actual table and then create a new describer
		// if a table has no action or an action doesn't refer to a table, them won't be added to result

		for _, ac := range actions_descr {

			if contains_int(actions_ids, ac.ActionId) {

				result = append(result, RuleDescriber{
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
	resInByte, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	res := string(resInByte)

	return &res
}

// Returns a describer for an already defined rule, basing the research on ActionName and TableName
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

// Returns pattern if the field respects a known one, using that parsers can know how to properly parse the field
func findIfKnownPattern(name string, bitwidth int) string {
	if strings.Contains(strings.ToLower(name), "port") {
		return pattern_port
	}
	if strings.Contains(strings.ToLower(name), "addr") {
		switch bitwidth {
		case 32:
			return pattern_ipv4_addr
		case 48:
			return pattern_mac_addr
		}
	}
	return ""
}

// util function, check if an array of integer contains a value
func contains_int(array []int, value int) bool {
	for _, el := range array {
		if el == value {
			return true
		}
	}
	return false
}