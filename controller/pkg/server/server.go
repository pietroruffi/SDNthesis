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
)

type SwitchServerData struct {
	Name           string
	ProgramName    string
	ProgramActions []p4switch.RuleDescriber
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
	http.HandleFunc("/executeProgram", executeProgram)
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir(serverPath+"web"))))

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")

	// TO-DO read available programs names
	// and when doing this PARSE the .p4 in .p4.p4info.json
	// REMEMBER!!
	programNames := []string{"simple", "simple1", "asymmetric"}

	// TO-DO read available switches

	var swData []SwitchServerData

	for _, sw := range allSwitches {
		swData = append(swData, SwitchServerData{
			Name:           sw.GetName(),
			ProgramName:    sw.GetProgramName(),
			ProgramActions: getDescribersForProgram(sw.GetProgramName()),
		})
	}

	data := RootPageData{
		Switches:       swData,
		ProgramNames:   programNames,
		SuccessMessage: successMessage,
		ErrorMessage:   errorMessage,
	}

	tmpl := template.Must(template.ParseFiles(serverPath + "indexGo.html"))

	err := tmpl.Execute(w, data)

	if err != nil {
		fmt.Println(err)
	}

	successMessage = ""
	errorMessage = ""
}

func addRule(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /addRule request")

	sw := r.URL.Query().Get("switch")

	idAction, err := strconv.Atoi(r.URL.Query().Get("idAction"))

	idTable, err := strconv.Atoi(r.URL.Query().Get("idTable"))

	// Questo codice estrae le informazioni dalle POST

	/*
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				fmt.Println("ParseForm() err:", err)
				return
			}

			// TO-DO handle this request:
			// 1) extract informations of actual rule
			// 2) add new rule to switch sw
			// 3) write a success/failure message on right variable
			// 4) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

			// REMEMBER add the program in execution on switch sw
			action := findActionByIdAndTable("asymmetric", idAction, idTable)

			var inputKeys []string

			for _, key := range action.Table.Keys {
				inputKeys = append(inputKeys, r.FormValue("key"+strconv.Itoa(key.Id)))
			}

			var inputParam []string
			for _, par := range action.Parameters {
				inputParam = append(inputParam, r.FormValue("par"+strconv.Itoa(par.Id)))
			}

			successMessage = "You successfully clicked on Add, good job!"

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}*/

	if r.Method == "GET" {
		data := AddRulePageData{
			SwitchName: sw,
			// REMEMBER add the program in execution on switch sw
			Rule: *findActionByIdAndTable("asymmetric", idAction, idTable),
		}

		tmpl := template.Must(template.ParseFiles(serverPath + "addRuleGo.html"))

		err = tmpl.Execute(w, data)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func executeProgram(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /executeProgram request")

	// TO-DO handle this request:
	// 1) change program in execution on switch
	// 2) write a success/failure message on right variable
	// 3) show index page by calling http.Redirect(w, r, "/", http.StatusSeeOther)

	successMessage = "You successfully clicked on Execute, good job!"

	http.Redirect(w, r, "/", http.StatusSeeOther)

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
