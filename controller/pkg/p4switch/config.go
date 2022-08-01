package p4switch

import (
	"fmt"
	"io/ioutil"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	p4InfoExt = ".p4info.txt"
	p4BinExt  = ".json"
	p4Path    = "../p4/"
)

func (sw *GrpcSwitch) ChangeConfig(newConfig *SwitchConfig) error {

	sw.config = newConfig

	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.InitiateConfig()
	sw.EnableDigest()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) ChangeConfigFile(configName string) error {
	sw.config = nil
	sw.initialConfigName = configName
	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.InitiateConfig()
	sw.EnableDigest()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) ChangeConfigFileSync(configName string) error {
	sw.config = nil
	sw.initialConfigName = configName
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.InitiateConfig()
	sw.EnableDigest()
	return nil
}

func (sw *GrpcSwitch) AddTableEntry(entry *p4_v1.TableEntry) error {
	if err := sw.p4RtC.InsertTableEntry(entry); err != nil { // InsertTableEntry in client/tables.go, sfrutta API di p4_v1 per inserire l'entry
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return err
	}
	sw.log.Tracef("Added entry: %+v", entry)

	return nil
}

func (sw *GrpcSwitch) RemoveTableEntry(entry *p4_v1.TableEntry) error {
	if err := sw.p4RtC.DeleteTableEntry(entry); err != nil { // DeleteTableEntry in client/tables.go, sfrutta API di p4_v1 per rimuovere l'entry
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return err
	}
	sw.log.Tracef("Added entry: %+v", entry)

	return nil
}

// TODO write better
// getConfig returns the configuration of the switch, if for some reasons config is nil, tries to read the configuration from file .yml and then returns, if also
// this try fails, return error
func (sw *GrpcSwitch) GetConfig() (*SwitchConfig, error) {
	if sw.config == nil {
		config, err := parseSwConfig(sw.GetName(), sw.initialConfigName)
		if err != nil {
			return nil, err
		}
		sw.config = config
	}
	return sw.config, nil
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

// TODO write different
// Read config of switch from the initiaConfigfile .yml and put it in the SwitchConfig field, then add the rules
func (sw *GrpcSwitch) InitiateConfig() error {
	config, err := sw.GetConfig()
	if err != nil {
		return err
	}
	for _, rule := range config.Rules {
		entry, err := CreateTableEntry(sw, rule)
		if err != nil {
			return err
		}
		sw.AddTableEntry(entry)
	}
	return nil
}

func readFileBytes(filePath string) []byte {
	var bytes []byte
	if filePath != "" {
		var err error
		if bytes, err = ioutil.ReadFile(filePath); err != nil {
			log.Fatalf("Error when reading binary from '%s': %v", filePath, err)
		}
	}
	return bytes
}

func (sw *GrpcSwitch) readP4Info() []byte {
	p4Info := p4Path + sw.GetProgramName() + p4InfoExt
	sw.log.Tracef("p4Info %s", p4Info)
	return readFileBytes(p4Info)
}

func (sw *GrpcSwitch) readBin() []byte {
	p4Bin := p4Path + sw.GetProgramName() + p4BinExt
	sw.log.Tracef("p4Bin %s", p4Bin)
	return readFileBytes(p4Bin)
}
