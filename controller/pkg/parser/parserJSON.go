package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Table struct {
	Id         int
	Name       string
	Keys       []Key
	ActionsIds []int
}

type Key struct {
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
	Name     string
	Bitwidth int
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

func getActions(nameProgram string) []Action {
	filename := path + nameProgram + ext

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

	// Extract tables informations

	var tables []Table

	for i := range result["pipelines"].([]interface{}) {

		all_tables := ((result["pipelines"].([]interface{})[i]).(map[string]interface{}))["tables"]

		for index_tables := range all_tables.([]interface{}) {

			table := all_tables.([]interface{})[index_tables].(map[string]interface{})

			// doesn't consider default tables (ones which starts with tbl_nameP4Program)
			if strings.HasPrefix(table["name"].(string), "tbl_"+nameProgram) {
				continue
			}

			var talbe_keys []Key

			for index_keys := range table["key"].([]interface{}) {

				key := table["key"].([]interface{})[index_keys].(map[string]interface{})

				// mask can either be present or not
				var mask string
				if key["mask"] != nil {
					mask = key["mask"].(string)
				}

				talbe_keys = append(talbe_keys, Key{
					Name:     key["name"].(string),
					Match:    key["match_type"].(string),
					Bitwidth: 32,
					//Target: c["target"].([]string), // add? useful?
					Mask: mask,
				})
			}

			var actions_ids []int
			for _, action_id := range table["action_ids"].([]interface{}) {
				actions_ids = append(actions_ids, int(action_id.(float64)))
			}

			tables = append(tables, Table{
				Id:         int(table["id"].(float64)),
				Name:       table["name"].(string),
				Keys:       talbe_keys,
				ActionsIds: actions_ids,
			})

		}

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
		if strings.HasPrefix(action["name"].(string), nameProgram) {
			continue
		}

		var action_parameters []Parameter

		for index_parameters := range action["runtime_data"].([]interface{}) {
			parameter := action["runtime_data"].([]interface{})[index_parameters].(map[string]interface{})

			action_parameters = append(action_parameters, Parameter{
				Name:     parameter["name"].(string),
				Bitwidth: int(parameter["bitwidth"].(float64)),
			})
		}
		id_action := int(action["id"].(float64))

		// find table which contains actual action
		var action_table Table
		for _, tab := range tables {
			if integer_contains(tab.ActionsIds, id_action) {
				action_table = tab
				break
			}
		}

		actions = append(actions, Action{
			Table:      action_table,
			Id:         id_action,
			Name:       action["name"].(string),
			Parameters: action_parameters,
		})
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

	headerFile, err := os.Open("header.html")

	if err != nil {
		fmt.Println(err)
	}
	fmt.Print("[DEBUG] Successfully Opened header.html", "\n\n")

	byteValue, _ := ioutil.ReadAll(headerFile)

	fmt.Fprintf(w, string(byteValue))

	headerFile.Close()

	fmt.Fprintf(w, "<div class='d-flex flex-column container-fluid align-items-center mt-5 mb-5'>\n")

	// Eventual error message
	if errorMessage != "" {
		fmt.Fprintf(w, "<div class='alert alert-danger mt-5' role='alert'>%s</div>", errorMessage)
		errorMessage = ""
	}
	//

	// Eventual success message
	if successMessage != "" {
		fmt.Fprintf(w, "<div class='alert alert-success mt-5' role='alert'>%s</div>", successMessage)
		successMessage = ""
	}

	// some <div> for better graphic ...
	fmt.Fprintf(w, "<div class='col-12 row justify-content-center mt-5'>\n")
	fmt.Fprintf(w, "<div class='col-12 col-lg-4 col-xl-3 justify-content-center align-items-center'>\n")

	// Part of web page where user can change the P4 program in execution on every switch
	fmt.Fprintf(w, "<h2 class='mb-3'>Change P4 program</h2>\n")

	// TO-DO read which program is actually in execution on every switch
	programOnSwitches := [3]string{"simple", "simple", "asymmetric"}

	// TO-DO read all available programs
	programs := [3]string{"simple", "simple1", "asymmetric"}

	// TO-DO change this for-loop for every switch
	for sw := 1; sw <= 3; sw++ {
		// print button where user can choose the switch
		fmt.Fprintf(w, "<h2 class='mt-4'><a class='btn btn-light fs-4 p-2 w-100' style='font-weight: 500' href='#switchS%dExec' data-bs-toggle='collapse'>Switch S%d</a></h2>\n", sw, sw)
		// print content of swich, i.e. programs in execution and which can be executed
		fmt.Fprintf(w, "<ul class='list-group collapse' id='switchS%dExec'>\n", sw)
		for _, prog := range programs {
			if prog == programOnSwitches[sw-1] {
				// if the program is in execution, the button Execute is disabled
				fmt.Fprintf(w, "<li class='list-group-item d-flex justify-content-between align-items-center'>%s\n", prog)
				fmt.Fprintf(w, "<button class='btn btn-success rounded-pill' disabled> Executing </button>\n</li>")
			} else {
				fmt.Fprintf(w, "<li class='list-group-item d-flex justify-content-between align-items-center'>%s\n", prog)
				fmt.Fprintf(w, "<a href='executeProgram?switch=s%d&program=%s'><button class='btn btn-primary rounded-pill'> Execute </button></a>\n</li>", sw, prog)
			}
		}
		fmt.Fprintf(w, "</ul>\n")
	}
	fmt.Fprintf(w, "</div>\n")
	fmt.Fprintf(w, "<div class='col-12 col-sm-6 col-lg-4 col-xl-3 mt-5 mt-lg-0 align-items-center'>\n")

	// Part of web page where user can add new rules (actions) on every switch
	fmt.Fprintf(w, "<h2 class='mb-3'>Install new rules</h2>\n")

	// TO-DO change this for-loop for every switch
	for sw := 1; sw <= 3; sw++ {
		// print button where user can choose the switch
		fmt.Fprintf(w, "<h2 class='mt-4'> <a class='btn btn-light fs-4 p-2 w-100' style='font-weight: 500' href='#switchS%dRule' data-bs-toggle='collapse'>Switch S%d</a> </h2>\n", sw, sw)

		fmt.Fprintf(w, "<div class='list-group collapse' id='switchS%dRule'>\n", sw)
		fmt.Fprintf(w, "<div class='accordion' id='accordionSwitchS%d'>\n", sw)

		// print actions information, for every action inside the P4 program actually executing on switch
		for index_rule, rule := range getActionsByP4InfoJson(programOnSwitches[sw-1]) {

			// beginning of new action
			fmt.Fprintf(w, "<div class='accordion-item'>")
			// button (actually is a button inside a h2 title) where user can choose to see informations of action
			fmt.Fprintf(w, "<h2 class='accordion-header' id='headingS%dRule%d'>\n", sw, index_rule)
			fmt.Fprintf(w, "<button class='accordion-button collapsed' type='button' data-bs-toggle='collapse' data-bs-target='#collapseS%dRule%d' aria-expanded='false' aria-controls='collapseS%dRule%d'>%s</button>\n", sw, index_rule, sw, index_rule, rule.Name)
			fmt.Fprintf(w, "</h2>\n")

			// informations inside the collapsed <div>
			fmt.Fprintf(w, "<div id='collapseS%dRule%d' class='accordion-collapse collapse' aria-labelledby='headingS%dRule%d' data-bs-parent='#accordionSwitchS%d'>\n<div class='accordion-body'>\n", sw, index_rule, sw, index_rule, sw)

			// Table name (with a little logo :D )
			fmt.Fprintf(w, "<strong>Table</strong>&nbsp;<img src='./web/img/table.svg'><br>\n")
			fmt.Fprintf(w, "<ul><li>%s</li></ul>\n", rule.Table.Name)

			// Keys of table (with a little logo :D )
			fmt.Fprintf(w, "<strong>Key</strong>&nbsp;<img src='./web/img/key.svg'><br>\n<ul>\n")
			// Just an img of a key
			for _, key := range rule.Table.Keys {
				fmt.Fprintf(w, "<li>%s (bit&lt;%d&gt;), match: %s</li>\n", key.Name, key.Bitwidth, key.Match)
			}
			fmt.Fprintf(w, "</ul>")

			// Parameters of action, if none print: "No parameter required"
			if len(rule.Parameters) > 0 {
				fmt.Fprintf(w, "<strong>Parameters</strong>&nbsp;<img src='./web/img/card-list.svg'><br>\n<ul>\n")
				for _, par := range rule.Parameters {
					fmt.Fprintf(w, "<li>%s (bit&lt;%d&gt;)</li>\n", par.Name, par.Bitwidth)
				}
				fmt.Fprintf(w, "</ul>")
			} else {
				fmt.Fprintf(w, "<strong>No parameter required</strong><br><br>\n")
			}

			// button which redirects to page /addRule where user can add the action he choose
			fmt.Fprintf(w, "<a href='addRule?switch=s%d&idRule=%d&idTable=%d'><button class='btn btn-primary rounded-pill' type='submit'>Add new rule</button></a>", sw, rule.Id, rule.Table.Id)
			fmt.Fprintf(w, "</div> </div> </div>")
		}
		fmt.Fprintf(w, "</div> </div>")
	}
	fmt.Fprintf(w, "</div> </div> </div> </body> </html>")
}

func addRule(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /addRule request")

	/*
		// Questo codice estrae le informazioni dalle POST

			if err := r.ParseForm(); err != nil {
				fmt.Fprintf(w, "ParseForm() err: %v", err)
				return
			}
			fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", r.PostForm)
			name := r.FormValue("name")
			address := r.FormValue("address")

		// CHANGE? Una volta finito rimandare alla pagina principale? La gestisco io la POST o qualcun altro?
	*/

	sw := r.URL.Query().Get("switch")

	idRule, err := strconv.Atoi(r.URL.Query().Get("idRule"))

	idTable, err := strconv.Atoi(r.URL.Query().Get("idTable"))

	headerFile, err := os.Open("header.html")

	if err != nil {
		fmt.Println(err)
	}
	fmt.Print("[DEBUG] Successfully Opened header.html", "\n\n")

	byteValue, _ := ioutil.ReadAll(headerFile)

	fmt.Fprintf(w, string(byteValue))

	headerFile.Close()
	// TO-DO: add which program is executing on switch
	action := findActionByIdAndTable("asymmetric", idRule, idTable)

	// some <div> for better graphic ...
	fmt.Fprintf(w, "<div class='d-flex flex-column container-fluid vh-100 justify-content-center align-items-center'>\n")
	fmt.Fprintf(w, "<div class='col-4 row justify-content-center align-items-center'>\n")

	// CHANGE? DON'T SANITIZE INPUT? WHICH PAGE HANDLE THE POST
	fmt.Fprintf(w, "<form class='col-12 row justify-content-center' action='/addRule?switch=%s&idRule=%d' method='POST'>\n", sw, idRule)

	// title
	fmt.Fprintf(w, "<h2>Add new rule for switch %s</h2>\n", sw)

	fmt.Fprintf(w, "<div class='mb-3'>\n")

	// show table name in textArea not modificable
	fmt.Fprintf(w, "<div class='mb-3 row'>\n")
	fmt.Fprintf(w, "<label for='staticTable' class='col-sm-2 col-form-label'><strong> Table </strong></label>\n")
	fmt.Fprintf(w, "<div class='col-sm-10'><input type='text' readonly class='form-control-plaintext' id='staticTable' value='%s'></div></div>\n", action.Table.Name)

	// show action name in textArea not modificable
	fmt.Fprintf(w, "<div class='mb-3 row'>\n")
	fmt.Fprintf(w, "<label for='staticRule' class='col-sm-2 col-form-label'><strong> Rule </strong></label>\n")
	fmt.Fprintf(w, "<div class='col-sm-10'><input type='text' readonly class='form-control-plaintext' id='staticRule' value='%s'></div></div>\n", action.Name)

	// show textInput where user can add keys
	for index, key := range action.Table.Keys {
		fmt.Fprintf(w, "<div class='mb-3 row'>\n")

		if index == 0 {
			// if first show the label "Key", than the list of keys
			fmt.Fprintf(w, "<label class='form-label'><strong> Key </strong></label>\n")
		}

		// information of key (name, bitwidth)
		fmt.Fprintf(w, "<label for='key%s' class='col-sm-5 col-form-label'> %s (bit&lt;%d&gt;)</label>\n", key.Name, key.Name, key.Bitwidth)

		// area where write the key
		fmt.Fprintf(w, "<div class='col-sm-7'><input type='text' class='form-control' name='key%s' id='key%s'></div>\n", key.Name, key.Name)
		// i use "key"+key.Name to reference the key because when i handle the POST request i have to be able to find the key (actually keys don't have id)

		fmt.Fprintf(w, "</div>\n")
	}

	// show textInput where user can add parameters (if there's no parameter just don't show anything)
	for index, par := range action.Parameters {
		fmt.Fprintf(w, "<div class='mb-3 row'>\n")

		if index == 0 {
			// if first show the label "Parameters", than the list of Parameters
			fmt.Fprintf(w, "<label class='form-label'><strong> Parameters </strong></label>\n")
		}
		// information of parameter (name, bitwidth)
		fmt.Fprintf(w, "<label for='par%s' class='col-sm-5 col-form-label'> %s (bit&lt;%d&gt;)</label>\n", par.Name, par.Name, par.Bitwidth)

		fmt.Fprintf(w, "<div class='col-sm-7'><input type='text' class='form-control' name='par%s' id='par%s'></div>\n", par.Name, par.Name)
		// as in keys i use "par"+parameter.Name to reference the parameter

		fmt.Fprintf(w, "</div>\n")
	}

	// submit button
	fmt.Fprintf(w, "<button type='submit' class='w-100 btn btn-success mt-3'>Add</button>\n")

	fmt.Fprintf(w, "</div> </form> </div> </div> </body> </html>\n")

	//successMessage = "Hai cliccato con successo su aggiungi regola, bravo"

	//getRoot(w, r)
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
