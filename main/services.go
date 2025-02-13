package main

// Mockup of whats in NEST atm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

// Service represents each service configuration for a virtual machine.
type Service struct {
	// REQUIRED
	Port int `yaml:"port"` // The port the service is running on

	// OPTIONALS
	// // SERVICE DEPENDENT
	User     string `yaml:"user,omitempty"`       // The username of a user for a service
	Password string `yaml:"password,omitempty"`   // The password of a user for a service
	QFile    string `yaml:"query_file,omitempty"` // The query file for a service
	QDir     string `yaml:"query_dir,omitempty"`  // The query directory for a service
	// // ONLY FOR SQL
	DBName string `yaml:"dbname,omitempty"` // name of the sql database being used
	// // TRUE OPTIONAL
	Award   int  `yaml:"award,omitempty"`   // The awarded points for having a service up at scoring time
	Partial bool `yaml:"partial,omitempty"` // Whether or not partial points should be awarded
}

const (
	// Timeouts, miliseconds
	router_timeout = 750
	ftp_timeout    = 250
	ssh_timeout    = 250
	sql_timeout    = 250
	dns_timeout    = 500
	web_timeout    = 1500
)

var (
	// Get the random seed for any random operations
	randseed int
)

func Initalize() {
	// Set the random seed for any random operations
	rand.Seed((uint64)(time.Now().Unix()))
}

// ChooseRandomUser reads the file at `dir`, which contains lines
// formatted as "username:password", picks one user at random, and
// returns the parsed username and password.
func ChooseRandomUser(dir string) (string, string, error) {
	data, err := os.ReadFile(dir)
	if err != nil {
		return "", "", fmt.Errorf("could not read users file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	if len(validLines) == 0 {
		return "", "", fmt.Errorf("no valid 'username:password' lines in %s", dir)
	}

	randomIndex := rand.Intn(len(validLines))
	userLine := validLines[randomIndex]

	// Parse username and password
	parts := strings.SplitN(userLine, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid user format: %s", userLine)
	}
	username := parts[0]
	password := parts[1]

	return username, password, nil
}
