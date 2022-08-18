package p4switch

import (
	"controller/pkg/client"
	"fmt"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	//log "github.com/sirupsen/logrus"
)

type Rule struct {
	Table       string
	Keys        []Key
	Action      string
	ActionParam []string `yaml:"action_param"`
	Describer   *RuleDescriber
}

type Key struct {
	Value string
	Mask  string // optional, used in keys with ternary match
}

type RuleDescriber struct {
	TableName    string
	TableId      int
	Keys         []FieldDescriber
	ActionName   string
	ActionId     int
	ActionParams []FieldDescriber
}

type FieldDescriber struct {
	Name      string
	Bitwidth  int
	MatchType string // optional, used in keys
	Pattern   string // optional, if present the parser will use this to discriminate which function parses this field
}

type SwitchConfig struct {
	Rules   []Rule
	Program string
	Digest  []string
}

// Return rules actually installed into the switch, supposing controller is the only one who can add rules, so returns only rules added by controller
func (sw *GrpcSwitch) GetInstalledRules() []Rule {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return nil
	}
	return config.Rules
}

// Adds a new rule into the switch, both in the array containg the installed rules and in the switch sw
func (sw *GrpcSwitch) AddRule(rule Rule) error {
	entry, err := CreateTableEntry(sw, rule)
	if err != nil {
		return err
	}

	sw.AddTableEntry(entry)

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return err
	}
	config.Rules = append(config.Rules, rule)
	return nil
}

// Removes the rule at index "idx" from the switch, both from the array containg the installed rules and from the switch sw
func (sw *GrpcSwitch) RemoveRule(idx int) error {
	entry, err := CreateTableEntry(sw, sw.config.Rules[idx])
	if err != nil {
		return err
	}

	sw.RemoveTableEntry(entry)

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting config of switch: %v", err)
		return err
	}
	config.Rules = append(config.Rules[:idx], config.Rules[idx+1:]...)
	return nil
}

// Return name of P4 program actually executing into the switch
func (sw *GrpcSwitch) GetProgramName() string {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting program name: %v", err)
		return ""
	}
	return config.Program
}

func (sw *GrpcSwitch) GetDigests() []string {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting digest list: %v", err)
		return make([]string, 0)
	}
	return config.Digest
}

// Create a variable of type p4_v1.TableEntry, corrisponding to the rule given by argument.
// Uses funcions of parser.go in order to parse Keys and ActionParameters
func CreateTableEntry(sw *GrpcSwitch, rule Rule) (*p4_v1.TableEntry, error) {

	descr := getDescriberFor(sw, rule)
	if descr == nil {
		return nil, fmt.Errorf("Error getting describer for rule %+v", rule)
	}
	rule.Describer = descr

	interfaces := parseKeys(rule.Keys, rule.Describer.Keys)
	if interfaces == nil {
		return nil, fmt.Errorf("Error parsing keys of rule %+v", rule)
	}

	parserActParam := getParserForActionParams("default")
	actionParams := parserActParam.parse(rule.ActionParam, rule.Describer.ActionParams)
	if actionParams == nil {
		return nil, fmt.Errorf("Error parsing action parameters of rule %+v", rule)
	}

	return sw.p4RtC.NewTableEntry(
		rule.Table,
		interfaces,
		sw.p4RtC.NewTableActionDirect(rule.Action, actionParams),
		nil,
	), nil
}

// Util function, gets all the keys of a rule and returns the parsed MatchInterfaces
func parseKeys(keys []Key, describers []FieldDescriber) []client.MatchInterface {
	result := make([]client.MatchInterface, len(keys))
	for idx, key := range keys {
		parserMatch := getParserForKeys(describers[idx].MatchType)
		result[idx] = parserMatch.parse(key, describers[idx])
		if result[idx] == nil {
			return nil
		}
	}
	return result
}
