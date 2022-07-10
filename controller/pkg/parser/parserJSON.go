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
	Name  string
	Match string
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
	path      = "../../../p4/"
	p4Program = "asymmetric"
	ext       = ".json"
)

func main() {
	http.HandleFunc("/", getRoot)
	//http.HandleFunc("/hello", getHello)
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getActions() []Action {
	filename := path + p4Program + ext

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
			if strings.HasPrefix(table["name"].(string), "tbl_"+p4Program) {
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
					Name:  key["name"].(string),
					Match: key["match_type"].(string),
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
	for _, ta := range tables {
		fmt.Println("[DEBUG-TABLES]", ta)
	}
	fmt.Print("\n")

	// Extract actions informations

	var actions []Action

	for index_actions := range result["actions"].([]interface{}) {

		action := (result["actions"].([]interface{})[index_actions]).(map[string]interface{})

		// doesn't consider default tables (ones which starts with nameP4Program)
		if strings.HasPrefix(action["name"].(string), p4Program) {
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
	for ac := range actions {
		fmt.Println("[DEBUG-ACTIONS]", actions[ac])
	}
	return actions
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	/*p := &Page{Title: title}
	err := templates.ExecuteTemplate(w, "index.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}*/
	jsonFile, err := os.Open("index.html")

	if err != nil {
		fmt.Println(err)
	}
	fmt.Print("[DEBUG] Successfully Opened index.html", "\n\n")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	fmt.Fprintf(w, string(byteValue))
}

func integer_contains(array []int, content int) bool {
	for _, el := range array {
		if el == content {
			return true
		}
	}
	return false
}
