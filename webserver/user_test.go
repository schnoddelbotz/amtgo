package webserver

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/schnoddelbotz/amtgo/database"
)

func TestCreateUser(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "amtgo-web")
	if err != nil {
		fmt.Print("Error creating temp dir for DB")
		os.Exit(1)
	}
	database.DbFile = tempdir + "/test.db"
	database.DbDriver = "sqlite3"

	database.OpenDB()
	createUser("foo", "bar", "baz")

	if authUser("foo", "bli") {
		t.Error("Created user successfully authenticates with bad password")
	}
	if !authUser("foo", "baz") {
		t.Error("Created user cannot authenticate despite correct password")
	}

	//database.CloseDB() -- FIXME closing causes problems in webserver_test.go...
}
