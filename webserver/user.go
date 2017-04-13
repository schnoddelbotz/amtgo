package webserver

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/schnoddelbotz/amtgo/database"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/ssh/terminal"
)

// CreateUserDialog shows a terminal-based user creation dialog.
func CreateUserDialog() {
	username, fullname, password := credentials()
	// this happens from terminal, so DB is not open yet...
	database.OpenDB()
	createUser(username, fullname, password)
	database.CloseDB()
}

func createUser(username string, fullname string, password string) {
	c := 32
	salt := make([]byte, c)
	rand.Read(salt)
	dk, _ := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)

	var u database.User
	u.Name = username
	u.Fullname = fullname
	u.Password = hex.EncodeToString(dk)
	u.Passsalt = hex.EncodeToString(salt)
	u.IsEnabled = 1
	database.InsertUser(u)
}

func authUser(username string, password string) bool {
	dbUser := database.GetUser(username)
	if dbUser.Name != "" {
		salt, _ := hex.DecodeString(dbUser.Passsalt)
		dk, _ := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
		passHash := hex.EncodeToString(dk)

		if passHash == dbUser.Password {
			return true
		}
	}
	return false
}

func credentials() (string, string, string) {
	// from: http://stackoverflow.com/questions/2137357/getpasswd-functionality-in-go
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)

	fmt.Print("\nEnter Password again: ")
	bytePassword2, _ := terminal.ReadPassword(int(syscall.Stdin))
	password2 := string(bytePassword2)

	if password != password2 {
		fmt.Println("\nError: passwords did not match, please try again.")
		os.Exit(1)
	}

	fmt.Print("\nEnter Fullname: ")
	fullname, _ := reader.ReadString('\n')

	return strings.TrimSpace(username), strings.TrimSpace(fullname), strings.TrimSpace(password)
}
