package p4switch

import (
	"controller/pkg/client"
	"fmt"
	"io/ioutil"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	//log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Table       string
	Keys        []Key
	Action      string
	ActionParam []string `yaml:"action_param"`
	Describer   RuleDescriber
}

type RuleDescriber struct {
	TableName    string
	TableId      int
	Keys         []FieldDescriber
	ActionName   string
	ActionId     int
	ActionParams []FieldDescriber
}

type Key struct {
	Value string
	Mask  string // optional, used in keys with ternary match
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
	return sw.installedRules
}

func (sw *GrpcSwitch) AddToInstalledRules(rule Rule) {
	sw.installedRules = append(sw.installedRules, rule)
}

func (sw *GrpcSwitch) RemoveFromInstalledRules(idx int) {
	sw.installedRules = append(sw.installedRules[:idx], sw.installedRules[idx+1:]...)
}

func (sw *GrpcSwitch) GetProgramName() string {
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting program name: %v", err)
		return ""
	}
	return config.Program
}

func (sw *GrpcSwitch) GetDigests() []string {
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting digest list: %v", err)
		return make([]string, 0)
	}
	return config.Digest
}

func GetEntriesOfConfigFile(sw *GrpcSwitch) []Rule {
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return nil
	}
	return config.Rules
}

func GetAllTableEntries(sw *GrpcSwitch) []*p4_v1.TableEntry {
	var tableEntries []*p4_v1.TableEntry
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return tableEntries
	}
	for _, rule := range config.Rules {
		tableEntries = append(tableEntries, CreateTableEntry(sw, rule))
	}
	return tableEntries
}

func parseSwConfig(swName string, configFileName string) (*SwitchConfig, error) {
	configs := make(map[string]SwitchConfig)
	configFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(configFile, &configs); err != nil {
		return nil, err
	}
	config := configs[swName]
	if config.Program == "" {
		return nil, fmt.Errorf("switch config not found in file %s", configFileName)
	}
	return &config, nil
}

func CreateTableEntry(sw *GrpcSwitch, rule Rule) *p4_v1.TableEntry {
	rule.Describer = *getDescriberFor(sw.GetProgramName(), rule)
	parserActParam := getParserForActionParams("default")

	return sw.p4RtC.NewTableEntry(
		rule.Table,
		parseMatchInterfaces(rule.Keys, rule.Describer.Keys),
		sw.p4RtC.NewTableActionDirect(rule.Action, parserActParam.parse(rule.ActionParam, rule.Describer.ActionParams)),
		nil,
	)
}

func parseMatchInterfaces(keys []Key, describers []FieldDescriber) []client.MatchInterface {
	result := make([]client.MatchInterface, len(keys))
	for idx, key := range keys {
		parserMatch := getParserForMatchInterface(describers[idx].MatchType)
		result[idx] = parserMatch.parse(key, describers[idx])
	}
	return result
}
