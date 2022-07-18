package p4switch

import (
	"io/ioutil"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
)

const (
	p4InfoExt = ".p4info.txt"
	p4BinExt  = ".json"
	p4Path    = "../p4/"
)

func (sw *GrpcSwitch) ChangeConfig(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRules()
	sw.EnableDigest()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) ChangeConfigSync(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRules()
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
	if err := sw.p4RtC.DeleteTableEntry(entry); err != nil { // InsertTableEntry in client/tables.go, sfrutta API di p4_v1 per inserire l'entry
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return err
	}
	sw.log.Tracef("Added entry: %+v", entry)

	return nil
}

func (sw *GrpcSwitch) addRules() {
	entries := GetAllTableEntries(sw) // GetAllTableEntries in rules.go, legge le regole dal file di configurazione .yml
	for _, entry := range entries {
		sw.AddTableEntry(entry)
	}
	for _, entry := range GetEntriesOfConfigFile(sw) {
		sw.AddToInstalledRules(entry)
	}
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
