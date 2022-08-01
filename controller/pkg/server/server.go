package server

import (
	"controller/pkg/p4switch"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

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

	programNames := []string{"simple", "simple1", "asymmetric"}

	var swData []SwitchServerData

	for _, sw := range allSwitches {
		swData = append(swData, SwitchServerData{
			Name:           sw.GetName(),
			ProgramName:    sw.GetProgramName(),
			ProgramActions: getDescribersForSwitch(sw),
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
		log.Errorf(err.Error())
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
		// 1) extract informations of rule
		// 2) add new rule to switch sw
		// 3) write a success/failure message on right variable
		// 4) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

		actualSwitch := getSwitchByName(sw)
		rule_descr := findActionByIdAndTable(actualSwitch, idAction, idTable)

		var inputKeys []p4switch.Key
		var inputMask string
		for idx, desc := range rule_descr.Keys {
			if strings.ToUpper(desc.MatchType) == "TERNARY" {
				inputMask = r.FormValue("mask" + strconv.Itoa(idx))
			}
			inputKeys = append(inputKeys, p4switch.Key{
				Value: r.FormValue("key" + strconv.Itoa(idx)),
				Mask:  inputMask,
			})
		}

		var inputParam []string
		for idx := range rule_descr.ActionParams {
			inputParam = append(inputParam, r.FormValue("par"+strconv.Itoa(idx)))
		}

		rule := p4switch.Rule{
			Table:       rule_descr.TableName,
			Keys:        inputKeys,
			Action:      rule_descr.ActionName,
			ActionParam: inputParam,
		}

		res := actualSwitch.AddRule(rule)

		if res != nil {
			errorMessage = "Failed to add entry: " + res.Error()
		} else {
			successMessage = "Entry added with success"
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if r.Method == "GET" {
		actualSwitch := getSwitchByName(sw)

		data := AddRulePageData{
			SwitchName: sw,
			Rule:       *findActionByIdAndTable(actualSwitch, idAction, idTable),
		}

		tmpl := template.Must(template.ParseFiles(serverPath + "addRule.html"))

		err = tmpl.Execute(w, data)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func removeRule(w http.ResponseWriter, r *http.Request) {

	swName := r.URL.Query().Get("switch")

	sw := getSwitchByName(swName)
	numRule, _ := strconv.Atoi(r.URL.Query().Get("number"))

	res := sw.RemoveRule(numRule)

	if res != nil {
		errorMessage = "Failed to delete entry: " + res.Error()
	} else {
		successMessage = "Entry deleted with success"
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func executeProgram(w http.ResponseWriter, r *http.Request) {

	// 1) change program in execution on switch
	// 2) write a success/failure message on right variable
	// 3) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

	program := r.URL.Query().Get("program")

	swName := r.URL.Query().Get("switch")

	sw := getSwitchByName(swName)

	err := sw.ChangeConfig(&p4switch.SwitchConfig{
		Program: program,
		Digest:  []string{},
		Rules:   []p4switch.Rule{},
	})

	if err != nil {
		errorMessage = "Cannot change configuration: " + err.Error()
	} else {
		successMessage = "Config updated to " + program
	}

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

func getDescribersForSwitch(sw *p4switch.GrpcSwitch) []p4switch.RuleDescriber {
	res := *p4switch.ParseP4Info(sw)

	var describers []p4switch.RuleDescriber

	json.Unmarshal([]byte(res), &describers)

	return describers
}

func findActionByIdAndTable(sw *p4switch.GrpcSwitch, idAction int, idTable int) *p4switch.RuleDescriber {
	for _, action := range getDescribersForSwitch(sw) {
		if action.ActionId == idAction && action.TableId == idTable {
			return &action
		}
	}
	return nil
}
