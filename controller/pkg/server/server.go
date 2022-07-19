package server

import (
	"context"
	"controller/pkg/p4switch"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

type SwitchServerData struct {
	Name           string
	ProgramName    string
	ProgramActions []p4switch.RuleDescriber
	InstalledRules []p4switch.Rule
}

type RootPageData struct {
	Switches       []SwitchServerData
	ProgramNames   []string
	ErrorMessage   string
	SuccessMessage string
}

type AddRulePageData struct {
	SwitchName string
	Rule       p4switch.RuleDescriber
}

const (
	pathJsonInfo = "../p4/JsonOfP4info/"
	extJsonInfo  = ".p4.p4info.json"
	serverPath   = "./pkg/server/"
)

var errorMessage string
var successMessage string

var allSwitches []*p4switch.GrpcSwitch

func StartServer(switches []*p4switch.GrpcSwitch) {
	allSwitches = switches

	http.HandleFunc("/", getRoot)
	http.HandleFunc("/addRule", addRule)
	http.HandleFunc("/removeRule", removeRule)
	http.HandleFunc("/executeProgram", executeProgram)

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir(serverPath+"web"))))

	log.Infof("Server listening on localhost:3333")
	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		log.Infof("server closed\n")
	} else if err != nil {
		log.Errorf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {

	// TO-DO read available programs names
	// and when doing this PARSE the .p4 in .p4.p4info.json

	programNames := []string{"simple", "simple1", "asymmetric"}

	var swData []SwitchServerData

	for _, sw := range allSwitches {
		swData = append(swData, SwitchServerData{
			Name:           sw.GetName(),
			ProgramName:    sw.GetProgramName(),
			ProgramActions: getDescribersForProgram(sw.GetProgramName()),
			InstalledRules: sw.GetInstalledRules(),
		})
	}

	data := RootPageData{
		Switches:       swData,
		ProgramNames:   programNames,
		SuccessMessage: successMessage,
		ErrorMessage:   errorMessage,
	}

	tmpl := template.Must(template.ParseFiles(serverPath + "index.html"))

	err := tmpl.Execute(w, data)

	if err != nil {
		fmt.Println(err)
	}

	successMessage = ""
	errorMessage = ""
}

func addRule(w http.ResponseWriter, r *http.Request) {

	sw := r.URL.Query().Get("switch")

	idAction, err := strconv.Atoi(r.URL.Query().Get("idAction"))

	idTable, err := strconv.Atoi(r.URL.Query().Get("idTable"))

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			fmt.Println("ParseForm() err:", err)
			return
		}

		// To handle this request:
		// 1) extract informations of actual rule
		// 2) add new rule to switch sw
		// 3) write a success/failure message on right variable
		// 4) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

		actualSwitch := getSwitchByName(sw)
		rule_descr := findActionByIdAndTable(actualSwitch.GetProgramName(), idAction, idTable)

		var inputKeys []string

		for idx := range rule_descr.Keys {
			inputKeys = append(inputKeys, r.FormValue("key"+strconv.Itoa(idx)))
		}

		var inputParam []string
		for idx := range rule_descr.ActionParams {
			inputParam = append(inputParam, r.FormValue("par"+strconv.Itoa(idx)))
		}

		rule := p4switch.Rule{
			Table:       rule_descr.TableName,
			Key:         inputKeys,
			Action:      rule_descr.ActionName,
			ActionParam: inputParam,
		}

		res := actualSwitch.AddTableEntry(p4switch.CreateTableEntry(actualSwitch, rule))

		if res != nil {
			errorMessage = "Failed to add entry: " + res.Error()
		} else {
			successMessage = "Entry added with success"
			actualSwitch.AddToInstalledRules(rule)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if r.Method == "GET" {
		actualSwitch := getSwitchByName(sw)

		data := AddRulePageData{
			SwitchName: sw,
			Rule:       *findActionByIdAndTable(actualSwitch.GetProgramName(), idAction, idTable),
		}

		tmpl := template.Must(template.ParseFiles(serverPath + "addRule.html"))

		err = tmpl.Execute(w, data)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func removeRule(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /removeRule request")

	swName := r.URL.Query().Get("switch")

	sw := getSwitchByName(swName)
	numRule, _ := strconv.Atoi(r.URL.Query().Get("number"))

	toDelete := sw.GetInstalledRules()[numRule]

	res := sw.RemoveTableEntry(p4switch.CreateTableEntry(sw, toDelete))

	if res != nil {
		errorMessage = "Failed to delete entry: " + res.Error()
	} else {
		successMessage = "Entry deleted with success"
		sw.RemoveFromInstalledRules(numRule)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func executeProgram(w http.ResponseWriter, r *http.Request) {

	// TO-DO handle this request:
	// 1) change program in execution on switch
	// 2) write a success/failure message on right variable
	// 3) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

	ctx, cancel := context.WithCancel(context.Background())

	swName := r.URL.Query().Get("switch")

	sw := getSwitchByName(swName)
	configName := r.URL.Query().Get("program")

	if err := sw.ChangeConfig(configName); err != nil { //ChangeConfig in p4switch/config.go
		if status.Convert(err).Code() == codes.Canceled {
			sw.GetLogger().Warn("Failed to update config, restarting")
			errorMessage = "Failed to update config, restarting"
			if err := sw.RunSwitch(ctx); err != nil {
				sw.GetLogger().Errorf("Cannot start")
				sw.GetLogger().Errorf("%v", err)
			}
		} else {
			sw.GetLogger().Errorf("Error updating swConfig: %v", err)
		}
	}
	sw.GetLogger().Tracef("Config updated to %s, ", configName)
	cancel()

	successMessage = "Config updated to " + configName

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func getSwitchByName(name string) *p4switch.GrpcSwitch {
	for _, sw := range allSwitches {
		if sw.GetName() == name {
			return sw
		}
	}
	return nil
}

func getDescribersForProgram(p4ProgramName string) []p4switch.RuleDescriber {
	res := *p4switch.ParseP4Info(p4ProgramName)

	var describers []p4switch.RuleDescriber

	json.Unmarshal([]byte(res), &describers)

	return describers
}

func findActionByIdAndTable(program string, idAction int, idTable int) *p4switch.RuleDescriber {
	for _, action := range getDescribersForProgram(program) {
		if action.ActionId == idAction && action.TableId == idTable {
			return &action
		}
	}
	return nil
}
