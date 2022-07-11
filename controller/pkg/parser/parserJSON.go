package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	path = "../../../p4/"
	ext  = ".json"
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

	jsonFile, err := os.Open("header.html")

	if err != nil {
		fmt.Println(err)
	}
	fmt.Print("[DEBUG] Successfully Opened header.html", "\n\n")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	fmt.Fprintf(w, string(byteValue))

	fmt.Fprintf(w, "<div class='d-flex flex-column container-fluid align-items-center mt-5 mb-5'>\n")

	// Eventuale messaggio di errore
	if errorMessage != "" {
		fmt.Fprintf(w, "<div class='alert alert-danger mt-5' role='alert'>%s</div>", errorMessage)
		errorMessage = ""
	}
	//

	// Eventuale messaggio di successo
	if successMessage != "" {
		fmt.Fprintf(w, "<div class='alert alert-success mt-5' role='alert'>%s</div>", successMessage)
		successMessage = ""
	}

	fmt.Fprintf(w, "<div class='col-12 row justify-content-center mt-5'>\n")
	fmt.Fprintf(w, "<div class='col-12 col-lg-4 col-xl-3 justify-content-center align-items-center'>\n")
	fmt.Fprintf(w, "<h2 class='mb-3'>Change P4 program</h2>\n")

	programOnSwitches := [3]string{"simple", "simple", "asymmetric"}

	// Ora inizia la parte da ripetere per ogni switch, per quanto riguarda l'esecuzione dei programmi
	programs := [3]string{"simple", "simple1", "asymmetric"}
	for sw := 1; sw <= 3; sw++ {
		fmt.Fprintf(w, "<h2 class='mt-4'><a class='btn btn-light fs-4 p-2 w-100' style='font-weight: 500' href='#switchS%dExec' data-bs-toggle='collapse'>Switch S%d</a></h2>\n", sw, sw)
		fmt.Fprintf(w, "<ul class='list-group collapse' id='switchS%dExec'>\n", sw)
		for _, prog := range programs {
			if prog == programOnSwitches[sw-1] {
				fmt.Fprintf(w, "<li class='list-group-item d-flex justify-content-between align-items-center'>%s\n", prog)
				fmt.Fprintf(w, "<button class='btn btn-success rounded-pill' disabled> Executing </button>\n</li>")
			} else {
				fmt.Fprintf(w, "<li class='list-group-item d-flex justify-content-between align-items-center'>%s\n", prog)
				fmt.Fprintf(w, "<a href='executeProgram?sw=s%d&pr=%s'><button class='btn btn-primary rounded-pill'> Execute </button></a>\n</li>", sw, prog)
			}
		}
		fmt.Fprintf(w, "</ul>\n")
	}
	fmt.Fprintf(w, "</div>\n")
	fmt.Fprintf(w, "<div class='col-12 col-sm-6 col-lg-4 col-xl-3 mt-5 mt-lg-0 align-items-center'>\n")
	fmt.Fprintf(w, "<h2 class='mb-3'>Install new rules</h2>\n")

	// Ora inizia la parte da ripetere per ogni switch, per quanto riguarda l'inserimento di regole
	for sw := 1; sw <= 3; sw++ {
		fmt.Fprintf(w, "<h2 class='mt-4'> <a class='btn btn-light fs-4 p-2 w-100' style='font-weight: 500' href='#switchS%dRule' data-bs-toggle='collapse'>Switch S%d</a> </h2>\n", sw, sw)
		fmt.Fprintf(w, "<div class='list-group collapse' id='switchS%dRule'>\n", sw)
		fmt.Fprintf(w, "<div class='accordion' id='accordionSwitchS%d'>\n", sw)

		for n, rule := range getActions(programOnSwitches[sw-1]) {

			fmt.Fprintf(w, "<div class='accordion-item'>")
			fmt.Fprintf(w, "<h2 class='accordion-header' id='headingS%dRule%d'>\n", sw, n)
			fmt.Fprintf(w, "<button class='accordion-button collapsed' type='button' data-bs-toggle='collapse' data-bs-target='#collapseS%dRule%d' aria-expanded='false' aria-controls='collapseS%dRule%d'>%s</button>\n", sw, n, sw, n, rule.Name)
			fmt.Fprintf(w, "</h2>\n")
			fmt.Fprintf(w, "<div id='collapseS%dRule%d' class='accordion-collapse collapse' aria-labelledby='headingS%dRule%d' data-bs-parent='#accordionSwitchS%d'>\n<div class='accordion-body'>\n", sw, n, sw, n, sw)

			fmt.Fprintf(w, "<strong>Table</strong><br>\n")
			fmt.Fprintf(w, "<ul><li>%s</li></ul>\n", rule.Table.Name)

			fmt.Fprintf(w, "<strong>Key</strong><br>\n<ul>\n")

			for _, key := range rule.Table.Keys {
				fmt.Fprintf(w, "<li>%s (bit&lt;%d&gt;), match: %s</li>\n", key.Name, key.Bitwidth, key.Match)
			}

			fmt.Fprintf(w, "</ul>")
			if len(rule.Parameters) > 0 {
				fmt.Fprintf(w, "<strong>Parameters</strong><br>\n<ul>\n")
				for _, par := range rule.Parameters {
					fmt.Fprintf(w, "<li>%s (bit&lt;%d&gt;)</li>\n", par.Name, par.Bitwidth)
				}
				fmt.Fprintf(w, "</ul>")
			} else {
				fmt.Fprintf(w, "<strong>No parameter required</strong><br><br>\n")
			}
			fmt.Fprintf(w, "<a href='addRule?switch=s%d&idTable=%d&idRule=%d'><button class='btn btn-primary rounded-pill' type='submit'>Add new rule</button></a>", sw, rule.Table.Id, rule.Id)
			fmt.Fprintf(w, "</div> </div> </div>")
		}
		fmt.Fprintf(w, "</div> </div>")
	}
	fmt.Fprintf(w, "</div> </div> </div> </body> </html>")
}

func addRule(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /addRule request")
	// TO-DO
	successMessage = "Hai cliccato con successo su aggiungi regola, bravo"
	getRoot(w, r)
}

func executeProgram(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /executeProgram request")
	// TO-DO
	errorMessage = "Hai cliccato con successo su esegui programma, bravo ma voglio testare l'errore"
	getRoot(w, r)
}

func integer_contains(array []int, content int) bool {
	for _, el := range array {
		if el == content {
			return true
		}
	}
	return false
}
