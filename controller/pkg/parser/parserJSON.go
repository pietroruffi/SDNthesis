package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Table struct {
	Id      int
	Name    string
	Keys    []Key
	Actions []Action
}

type Key struct {
	Name   string
	Match  string
	Mask   string
	Target []string
}

type Action struct {
	Id         int
	Name       string
	Parameters []Parameter
	//  Table      Table (mettere ?)
}

type Parameter struct {
	Name     string
	Bitwidth int
}

func main() {

	filename := "../../../p4/simple.json"
	// Open our jsonFile
	jsonFile, err := os.Open(filename)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print("Successfully Opened", filename, "\n\n")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}
	json.Unmarshal([]byte(byteValue), &result)

	// Extract tables informations

	for i := range result["pipelines"].([]interface{}) {
		a := ((result["pipelines"].([]interface{})[i]).(map[string]interface{}))["tables"]

		for k := range a.([]interface{}) {
			b := a.([]interface{})[k].(map[string]interface{})
			fmt.Println(b["id"], b["name"], b["key"], b["action_ids"], b["actions"])
		}

	}

	fmt.Print("\n")

	// Extract actions informations

	for i := range result["actions"].([]interface{}) {
		a := (result["actions"].([]interface{})[i]).(map[string]interface{})
		fmt.Println(a["id"], a["name"])
		for k := range a["runtime_data"].([]interface{}) {
			fmt.Println("\t", a["runtime_data"].([]interface{})[k])
		}
	}

}
