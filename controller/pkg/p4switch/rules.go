package p4switch

import (
	"controller/pkg/client"

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

func (sw *GrpcSwitch) GetInstalledRules() []Rule {
	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return nil
	}
	return config.Rules
}

func (sw *GrpcSwitch) AddRule(rule Rule) error {
	entry, err := CreateTableEntry(sw, rule)
	if err != nil {
		return err
	}

	sw.AddTableEntry(entry)

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return err
	}
	config.Rules = append(config.Rules, rule)
	return nil
}

func (sw *GrpcSwitch) RemoveRule(idx int) error {
	entry, err := CreateTableEntry(sw, sw.config.Rules[idx])
	if err != nil {
		return err
	}

	sw.RemoveTableEntry(entry)

	config, err := sw.GetConfig()
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return err
	}
	config.Rules = append(config.Rules[:idx], config.Rules[idx+1:]...)
	return nil
}

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

func CreateTableEntry(sw *GrpcSwitch, rule Rule) (*p4_v1.TableEntry, error) {
	// TODO ADD ERRORS
	rule.Describer = getDescriberFor(sw.GetProgramName(), rule)
	parserActParam := getParserForActionParams("default")

	return sw.p4RtC.NewTableEntry(
		rule.Table,
		parseMatchInterfaces(rule.Keys, rule.Describer.Keys),
		sw.p4RtC.NewTableActionDirect(rule.Action, parserActParam.parse(rule.ActionParam, rule.Describer.ActionParams)),
		nil,
	), nil
}

func parseMatchInterfaces(keys []Key, describers []FieldDescriber) []client.MatchInterface {
	result := make([]client.MatchInterface, len(keys))
	for idx, key := range keys {
		parserMatch := getParserForMatchInterface(describers[idx].MatchType)
		result[idx] = parserMatch.parse(key, describers[idx])
	}
	return result
}
