package ansible

import (
	"bufio"
	"fmt"
	"os"
	"math/rand"
	"strings"
	"time"

	"github.com/fabric8io/gosupervise/log"
)

type HostEntry struct {
	Name       string
	Host       string
	User       string
	PrivateKey string
}

// ChooseHostAndPrivateKey parses the given Ansbile inventory file for the hosts
// and chooses a single host inside it, returning the host name and the private key
func ChooseHostAndPrivateKey(inventoryFile string, hosts string) (*HostEntry, error) {
	file, err := os.Open(inventoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hostEntries := []HostEntry{}
	hostsLine := "[" + hosts + "]"
	foundHeader := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) > 0 {
			if (foundHeader) {
				if text[0] == '[' {
					break
				} else {
					hostEntry := parseHostEntry(text)
					if hostEntry != nil {
						hostEntries = append(hostEntries, *hostEntry)
					}
				}
			} else if text == hostsLine {
				foundHeader = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	count := len(hostEntries)
	log.Info("Found %v host entries", count)

	// lets pick a random entry
	if count > 0 {
		pickedEntry := hostEntries[random(0, count)]
		if len(pickedEntry.Host) == 0 {
			return nil, fmt.Errorf("Could not find host name for entry %s", pickedEntry.Name)
		}
		if len(pickedEntry.PrivateKey) == 0 {
			return nil, fmt.Errorf("Could not find PrivateKey for entry %s", pickedEntry.Name)
		}
		if len(pickedEntry.User) == 0 {
			return nil, fmt.Errorf("Could not find User for entry %s", pickedEntry.Name)
		}
		log.Info("Picked host " + pickedEntry.Host)
		return &pickedEntry, nil
	}
	return nil, fmt.Errorf("Could not find any hosts for inventory file %s and hosts %s", inventoryFile, hosts)
}

func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func parseHostEntry(text string) *HostEntry {
	values := strings.Split(text, " ")
	name := ""
	user := ""
	host := ""
	privateKey := ""
	count := len(values)
	if count > 0 {
		name = values[0];

		// lets parse the key value expressions for the host name
		for _, exp := range values[1:] {
			params := strings.Split(exp, "=")
			if len(params) == 2 {
				paramValue := params[1]
				switch (params[0]) {
				case "ansible_ssh_host":
					host = paramValue
				case "ansible_ssh_user":
					user = paramValue
				case "ansible_ssh_private_key_file":
					privateKey = paramValue
				}
			}
		}

		// if there's no host defined yet, lets assume that the name is the host name
		if len(host) == 0 {
			host = name
		}
	}
	return &HostEntry{
		Name: name,
		Host: host,
		User: user,
		PrivateKey: privateKey,
	}
}