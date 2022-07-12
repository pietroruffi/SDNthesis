package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

type Table struct {
	Id         int
	Name       string
	Keys       []Key
	ActionsIds []int
}

type Key struct {
	Id       int
	Name     string
	Match    string
	Bitwidth int
	//Target []string // add? useful?
	Mask string
}

type Action struct {
	Table      Table
	Id         int
	Name       string
	Parameters []Parameter
}

type Parameter struct {
	Id       int
	Name     string
	Bitwidth int
}

type Switch struct {
	Name    string
	Program Program
}

type Program struct {
	Name    string
	Actions []Action
}

type RootPageData struct {
	Switches     []Switch
	ProgramNames []string
}

type AddRulePageData struct {
	SwitchName string
	Rule       Action
}

const (
	path         = "../../../p4/"
	pathJsonInfo = "../../../p4/JsonOfP4info/"
	ext          = ".json"
	extJsonInfo  = ".p4.p4info.json"
)

var errorMessage string
var successMessage string

func main() {
	http.HandleFunc("/", getRoot)
	http.HandleFunc("/addRule", addRule)
	http.HandleFunc("/executeProgram", executeProgram)
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getActionsByP4InfoJson(nameProgram string) []Action {

	// TO-DO save actions in static variable so u don't have to read the files every time

	filename := pathJsonInfo + nameProgram + extJsonInfo

	jsonFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		return nil
	}
	fmt.Print("[DEBUG] Successfully Opened ", filename, "\n\n")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}

	json.Unmarshal([]byte(byteValue), &result)

	var tables []Table

	for i := range result["tables"].([]interface{}) {

		table := result["tables"].([]interface{})[i].(map[string]interface{})

		// doesn't consider default tables (ones which starts with tbl_nameP4Program)
		/*if strings.HasPrefix(table["name"].(string), "tbl_"+nameProgram) {
			continue
		}*/

		preamble := table["preamble"].(map[string]interface{})
		table_name := preamble["name"]
		table_id := int(preamble["id"].(float64))

		var talbe_keys []Key

		for index_keys := range table["matchFields"].([]interface{}) {

			key := table["matchFields"].([]interface{})[index_keys].(map[string]interface{})

			// mask can either be present or not
			var mask string
			if key["mask"] != nil {
				mask = key["mask"].(string)
			}

			talbe_keys = append(talbe_keys, Key{
				Id:       int(key["id"].(float64)),
				Name:     key["name"].(string),
				Match:    key["matchType"].(string),
				Bitwidth: int(key["bitwidth"].(float64)),
				//Target: c["target"].([]string), // add? useful?
				Mask: mask,
			})
		}

		var actions_ids []int
		for _, action_id := range table["actionRefs"].([]interface{}) {
			actions_ids = append(actions_ids, int(action_id.(map[string]interface{})["id"].(float64)))
		}

		tables = append(tables, Table{
			Id:         table_id,
			Name:       table_name.(string),
			Keys:       talbe_keys,
			ActionsIds: actions_ids,
		})
	}
	/*
		for _, ta := range tables {
			fmt.Println("[DEBUG-TABLES]", ta)
		}
		fmt.Print("\n")
	*/
	// Extract actions informations

	var actions []Action

	for index_actions := range result["actions"].([]interface{}) {

		action := (result["actions"].([]interface{})[index_actions]).(map[string]interface{})

		// doesn't consider default tables (ones which starts with nameP4Program)
		/*
			if strings.HasPrefix(action["name"].(string), nameProgram) {
				continue
			}
		*/

		preamble := action["preamble"].(map[string]interface{})
		action_name := preamble["name"]
		action_id := int(preamble["id"].(float64))

		//fmt.Print("\n\t", action, "\n\t", action_name, action_id, "\n\n")

		var action_parameters []Parameter

		if action["params"] != nil {
			for index_parameters := range action["params"].([]interface{}) {
				parameter := action["params"].([]interface{})[index_parameters].(map[string]interface{})

				action_parameters = append(action_parameters, Parameter{
					Id:       int(parameter["id"].(float64)),
					Name:     parameter["name"].(string),
					Bitwidth: int(parameter["bitwidth"].(float64)),
				})
			}
		}
		// find table which contains actual action

		for _, action_table := range tables {

			if integer_contains(action_table.ActionsIds, action_id) {

				actions = append(actions, Action{
					Table:      action_table,
					Id:         action_id,
					Name:       action_name.(string),
					Parameters: action_parameters,
				})

			}
		}

	}
	/*
		for ac := range actions {
			fmt.Println("[DEBUG-ACTIONS]", actions[ac])
		}
	*/
	return actions
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")

	programNames := []string{"simple", "simple1", "asymmetric"}

	var programs []Program

	for _, prog := range programNames {
		programs = append(programs, Program{
			Name:    prog,
			Actions: getActionsByP4InfoJson(prog),
		})
	}

	data := RootPageData{
		Switches: []Switch{
			{Name: "s1", Program: Program{
				Name:    "simple",
				Actions: getActionsByP4InfoJson("simple"),
			}},
			{Name: "s2", Program: Program{
				Name:    "asymmetric",
				Actions: getActionsByP4InfoJson("asymmetric"),
			}},
			{Name: "s3", Program: Program{
				Name:    "simple1",
				Actions: getActionsByP4InfoJson("simple1"),
			}},
		},
		ProgramNames: programNames,
	}

	tmpl := template.Must(template.ParseFiles("indexGo.html"))

	err := tmpl.Execute(w, data)

	if err != nil {
		fmt.Println(err)
	}
}

func addRule(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /addRule request")

	sw := r.URL.Query().Get("switch")

	idRule, err := strconv.Atoi(r.URL.Query().Get("idRule"))

	idTable, err := strconv.Atoi(r.URL.Query().Get("idTable"))

	// Questo codice estrae le informazioni dalle POST
	// CHANGE? Una volta finito rimandare alla pagina principale? La gestisco io la POST o qualcun altro?

	/*
		if err := r.ParseForm(); err != nil {
			fmt.Println("ParseForm() err:", err)
			return
		}
		fmt.Println("Post from website! r.PostFrom =", r.PostForm)
		key1 := r.FormValue("key1")
		par1 := r.FormValue("par1")
		par2 := r.FormValue("par2")

		fmt.Println(key1, par1, par2)
	*/
	data := AddRulePageData{
		SwitchName: sw,
		Rule:       *findActionByIdAndTable("asymmetric", idRule, idTable),
	}

	tmpl := template.Must(template.ParseFiles("addRuleGo.html"))

	err = tmpl.Execute(w, data)

	if err != nil {
		fmt.Println(err)
	}
}

func executeProgram(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /executeProgram request")

	// TO-DO handle this request:
	// 1) change program in execution on switch
	// 2) write a success/failure message on right variable
	// 3) show index page by calling getRoot()

	successMessage = "You successfully clicked on Execute, good job!"
	getRoot(w, r)
}

func findActionByIdAndTable(program string, idAction int, idTable int) *Action {
	for _, action := range getActionsByP4InfoJson(program) {
		if action.Id == idAction && action.Table.Id == idTable {
			return &action
		}
	}
	return nil
}

func integer_contains(array []int, content int) bool {
	for _, el := range array {
		if el == content {
			return true
		}
	}
	return false
}
