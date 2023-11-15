package onepassword

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type VaultEntries []VaultEntry

type VaultEntry struct {
	ID      string `json:"id,omitempty"`
	Title   string `json:"title,omitempty"`
	Version int    `json:"version,omitempty"`
	Vault   struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"vault,omitempty"`
	Category              string    `json:"category,omitempty"`
	LastEditedBy          string    `json:"last_edited_by,omitempty"`
	CreatedAt             time.Time `json:"created_at,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
	AdditionalInformation string    `json:"additional_information,omitempty"`
	Urls                  []struct {
		Label   string `json:"label,omitempty"`
		Primary bool   `json:"primary,omitempty"`
		Href    string `json:"href,omitempty"`
	} `json:"urls,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Favorite bool     `json:"favorite,omitempty"`
}

// GetMFAToken for alias from Vault Entries
func GetMFAToken(alias string, vaultEntries VaultEntries) (tokenAccountId string, token string, err error) {
	for _, entry := range vaultEntries {
		if entry.Title == alias {
			// get the password from 1Password
			//  # MFA read from screen
			//  otpuri=$( op read "op://Adobe/${id}/one-time password"  )
			cmd := exec.Command("sh", "-c", fmt.Sprintf(`op read "op://Adobe/%v/one-time password"`, entry.ID))
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()

			if err != nil {
				log.Printf("Error getting otp for %v from 1Password: %v", alias, err)
				log.Fatalf("Stderr: %v", string(stderr.Bytes()))
			}
			token = strings.TrimSpace(string(stdout.Bytes()))
			//  # Account ID from otp value
			//  accountId=$( echo ${otpuri}  | cut -d"@" -f2 | cut -d"?" -f1 )
			tokenAccountId = strings.Split(strings.Split(strings.TrimSpace(string(stdout.Bytes())), "@")[1], "?")[0]

			//   # Silly hack to decide whether the name of the token includes an @ symbol
			//  # if so, we have to get the third field instead of the second to get the
			//  # account ID and MFA from ...
			//  if [[ ${accountId} =~ ^[0-9]+$ ]]
			//  then
			//      mfa=$( echo ${otpuri} | cut -d"@" -f2 | cut -d"?" -f2 | cut -d"=" -f2 | cut -d"&" -f1 )
			//  else
			//      # TODO - figure out how to make this part smarter, as it currently relies on a specific format
			//      # and that there is only one @ symbol in the name
			//      accountId=$( echo ${otpuri}  | cut -d"@" -f3 | cut -d"?" -f1 )
			//      mfa=$( echo ${otpuri} | cut -d"@" -f3 | cut -d"?" -f2 | cut -d"=" -f2 | cut -d"&" -f1 )
			//  fi
			_, err = strconv.Atoi(tokenAccountId)
			if err == nil {
				// if it's a number, figure out the actual token part ...
				token = strings.Split(token, "@")[1]
				token = strings.Split(token, "?")[1]
				token = strings.Split(token, "=")[1]
				token = strings.Split(token, "&")[0]
			} else {
				// if it is a number, we have to get the second field instead of the first to get the account ID and MFA from
				tokenAccountId = strings.Split(token, "@")[2]
				tokenAccountId = strings.Split(tokenAccountId, "?")[0]
				// Now that we have the account ID, we can get the token
				token = strings.Split(token, "@")[2]
				token = strings.Split(token, "?")[1]
				token = strings.Split(token, "=")[1]
				token = strings.Split(token, "&")[0]
			}

			// don't bother returning error from above as we were calculating the token.
			return tokenAccountId, token, nil
		}
	}
	return
}

func VaultEntryHasMFA(entry VaultEntry) (hasMFA bool) {
	// get the MFA from 1Password
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`op item get %v --otp`, entry.ID))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	hasMFA = err == nil
	return hasMFA
}

func GetPassword(alias string, vaultEntries VaultEntries) (password string, err error) {
	for _, entry := range vaultEntries {
		if entry.Title == alias {
			// get the password from 1Password
			cmd := exec.Command("sh", "-c", fmt.Sprintf(`op read op://Adobe/%v/password`, entry.ID))
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()

			if err != nil {
				log.Printf("Error getting password for %v from 1Password: %v", alias, err)
				log.Fatalf("Stderr: %v", string(stderr.Bytes()))
			}
			password = strings.TrimSpace(string(stdout.Bytes()))
			return password, nil
		}
	}
	return
}

// GetVaultEntries - reads all the vault entries from a specific vault that have a specific tag
func GetVaultEntries(vault string, tags string) (vaultEntries VaultEntries) {

	myClient, err := NewClient(&ClientOptions{})
	if err != nil {
		log.Fatalf("Error getting client: %v", err)
	}
	err = myClient.GetJSON(&vaultEntries, "item", "list", "--tags", tags, "--vault", vault)
	if err != nil {
		log.Fatalf("Error getting vault entries")
	}

	// get the list of accounts from 1Password
	//	list=$( op item list --vault Adobe --format json |  jq -rc --arg firstOne "${firstOne}" --arg lastOne "${lastOne}" 'sort_by(.title) |.[] | .alias = (.title | split(" "))[0] | select(.alias >= $firstOne) | select(.alias <= $lastOne)| { "id": .id }' )
	//cmd := exec.Command("sh", "-c", `op item list --vault Adobe --tags AWS --format json`)
	//var stdout, stderr bytes.Buffer
	//cmd.Stdout = &stdout
	//cmd.Stderr = &stderr
	//err := cmd.Run()
	//
	//if err != nil {
	//	log.Printf("Error getting Adobe passwords from 1Password: %v", err)
	//	log.Fatalf("Stderr: %v", string(stderr.Bytes()))
	//}
	//
	//err = json.Unmarshal(stdout.Bytes(), &vaultEntries)
	//if err != nil {
	//	log.Fatalf("Error unmarshalling JSON: %v", err)
	//}
	//x, _ := json.MarshalIndent(vaultEntries, "", "  ")
	//fmt.Println(string(x))
	return vaultEntries
}

func AddToVaultIfNotFound(alias string, vaultEntries VaultEntries) (vaultEntry VaultEntry, found bool) {
	var stdout, stderr bytes.Buffer

	// find the account in the list
	for _, entry := range vaultEntries {
		if entry.Title == alias {
			return entry, true
		}
	}
	if !found {
		stdout.Reset()
		log.Printf("Account %#v not found in 1Password", alias)
		// Add the account to 1password
		cmd := exec.Command("sh", "-c", fmt.Sprintf(`op item create \
    --category login \
    --title %#[1]v \
    --vault Adobe \
    --url 'https://%[1]v.signin.aws.amazon.com/console' \
    --generate-password='letters,digits,symbols,30' \
    --tags Adobe,AWS \
	'username=adobe%[2]v@adobe.com' \
    --format json`,
			alias, strings.ToLower(alias)))

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			log.Printf("Error getting Adobe passwords from 1Password: %v", err)
			log.Fatalf("Stderr: %v", string(stderr.Bytes()))
		}
		err = json.Unmarshal(stdout.Bytes(), &vaultEntry)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON: %v", err)
		}
	}
	return vaultEntry, found
}
