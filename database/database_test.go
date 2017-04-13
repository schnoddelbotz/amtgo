package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/schnoddelbotz/amtgo/amt"
)

func init() {
	tempdir, err := ioutil.TempDir("", "amtgo-db")
	if err != nil {
		fmt.Print("Error creating temp dir for DB")
		os.Exit(1)
	}
	DbFile = tempdir + "/test.db"
	DbDriver = "sqlite3"

	OpenDB()
	// these reference users, which would interfere with testing...
	db.Exec("DELETE FROM notification")
	db.Exec("DELETE FROM job")
}

func TestUser(t *testing.T) {
	var u User
	u.Fullname = "John Doe"
	u.Name = "john"
	u.IsAdmin = 1
	u.OuID = 1
	u.Password = "abc"
	u.Passsalt = "cde"
	InsertUser(u)

	created := GetUser("john")
	if created.Fullname != "John Doe" {
		t.Error("User john wasn't created with desired name!")
	}

	DeleteUser(created.ID)
	deleted := GetUser("john")
	if deleted.Fullname == "John Doe" {
		t.Error("User john wasn't deleted as expected")
	}
}

func TestOu(t *testing.T) {
	// fake ember-data submission of OU data
	submitData := `{"ou":{"id":8,"parent_id":"1","optionset_id":"1","name":"MyRoom","description":"Descriptiontext","idle_power":123.45,"logging":true}}`
	data := ioutil.NopCloser(bytes.NewReader([]byte(submitData)))
	createdOuJSON := InsertOu(data)

	// convert JSON response to Ou
	type singleOu struct {
		Ou Ou `json:"ou"`
	}
	var myOu singleOu
	var ouJSON = []byte(createdOuJSON)
	err := json.Unmarshal(ouJSON, &myOu)
	if err != nil {
		t.Error("Failed to unmarshal JSON response for newly created OU")
	}
	if myOu.Ou.Name != "MyRoom" || myOu.Ou.IdlePower != 123.45 {
		t.Error("Submitted OU hasn't desired content")
	}

	// update Ou
	submitData = fmt.Sprintf(`{"ou":{"id":%d,"parent_id":"1","optionset_id":"1","name":"MyUpdatedRoom","description":"Descriptiontext","idle_power":78.9,"logging":true}}`, myOu.Ou.ID)
	data = ioutil.NopCloser(bytes.NewReader([]byte(submitData)))
	updateOuJSON := UpdateOu(myOu.Ou.ID, data)
	var updatedOu singleOu
	var updatedOuJSON = []byte(updateOuJSON)
	err = json.Unmarshal(updatedOuJSON, &updatedOu)
	if err != nil {
		t.Error("Failed to unmarshal JSON response for updated OU")
	}
	if updatedOu.Ou.Name != "MyUpdatedRoom" || updatedOu.Ou.IdlePower != 78.9 {
		t.Error("Updated OU hasn't desired content")
	}

	// delete Ou
	msg, deleteSuccess := DeleteOu(myOu.Ou.ID)
	if !deleteSuccess {
		t.Errorf("Deletion of OU failed: %s", msg)
	}

	// ensure deletion success
	testGetOu := GetOu(myOu.Ou.ID)
	if testGetOu.Name == "MyRoom" {
		t.Error("Deletion of OU was reported successful, but that was a lie")
	}
}

func TestOptionset(t *testing.T) {
	// fake ember-data submission of Optionset data
	submitData := `{"optionset":{"name":"TestOptionSet","description":"TestDescription","sw_dash":true,"sw_v5":false,"sw_scan22":true,"sw_scan3389":true,"sw_usetls":true,"sw_skipcertchk":false,"opt_timeout":"123","opt_passfile":"/tmp/pwfile","opt_cacertfile":"/tmp/cacertfile"}}`
	data := ioutil.NopCloser(bytes.NewReader([]byte(submitData)))
	createdOptionsetJSON := InsertOptionset(data)

	// convert JSON response to Optionset
	type singleOptionset struct {
		Optionset amt.Optionset `json:"optionset"`
	}
	var myOptionset singleOptionset
	var optionsetJSON = []byte(createdOptionsetJSON)
	err := json.Unmarshal(optionsetJSON, &myOptionset)
	if err != nil {
		t.Error("Failed to unmarshal JSON response for newly created Optionset")
	}
	testOptionset := myOptionset.Optionset
	if testOptionset.Name != "TestOptionSet" || testOptionset.Description != "TestDescription" {
		t.Error("Submitted Optionset hasn't desired content")
	}

	// update Optionset
	submitData = `{"optionset":{"name":"TestOptionSetXX","description":"TestDescriptionZZ","sw_dash":true,"sw_v5":false,"sw_scan22":true,"sw_scan3389":true,"sw_usetls":true,"sw_skipcertchk":false,"opt_timeout":"123","opt_passfile":"/tmp/pwfile","opt_cacertfile":"/tmp/cacertfile"}}`
	data = ioutil.NopCloser(bytes.NewReader([]byte(submitData)))
	updateOptionsetJSON := UpdateOptionset(testOptionset.ID, data)
	var updatedOptionset singleOptionset
	var updatedOptionsetJSON = []byte(updateOptionsetJSON)
	err = json.Unmarshal(updatedOptionsetJSON, &updatedOptionset)
	if err != nil {
		t.Error("Failed to unmarshal JSON response for updated Optionset")
	}
	if updatedOptionset.Optionset.Name != "TestOptionSetXX" || updatedOptionset.Optionset.Description != "TestDescriptionZZ" {
		t.Errorf("Updated Optionset hasn't desired content: %+v", updatedOptionset)
	}

	// delete Ou
	msg, deleteSuccess := DeleteOptionset(testOptionset.ID)
	if !deleteSuccess {
		t.Errorf("Deletion of OU failed: %s", msg)
	}

	// // ensure deletion success
	testGetOptionset := GetOptionset(testOptionset.ID)
	if testGetOptionset.Name == "TestOptionSetXX" {
		t.Error("Deletion of Optionset was reported successful, but that was a lie")
	}
}
