package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	//Target []string
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

	filename := path + p4Program + ext
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

	var tables []Table
	for i := range result["pipelines"].([]interface{}) {
		a := ((result["pipelines"].([]interface{})[i]).(map[string]interface{}))["tables"]

		for k := range a.([]interface{}) {
			b := a.([]interface{})[k].(map[string]interface{})

			if strings.HasPrefix(b["name"].(string), "tbl_"+p4Program) {
				continue
			}

			//fmt.Println(b["id"], b["name"], b["key"], b["action_ids"], b["actions"])
			id := b["id"].(float64)

			var keys []Key

			for kk := range b["key"].([]interface{}) {
				c := b["key"].([]interface{})[kk].(map[string]interface{})

				var mask string
				if c["mask"] != nil {
					mask = c["mask"].(string)
				}

				keys = append(keys, Key{
					Name:  c["name"].(string),
					Match: c["match_type"].(string),
					//Target: c["target"].([]string),
					Mask: mask,
				})
			}

			var action_ids []int
			for ii := range b["action_ids"].([]interface{}) {
				action_ids = append(action_ids, int(b["action_ids"].([]interface{})[ii].(float64)))
			}

			tables = append(tables, Table{
				Id:         int(id),
				Name:       b["name"].(string),
				Keys:       keys,
				ActionsIds: action_ids,
			})

		}

	}

	fmt.Println(tables)
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
