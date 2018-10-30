package jwt

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	JWT "github.com/dgrijalva/jwt-go"
	"os"
	"regexp"
)

func gen_jwt(_key *ecdsa.PrivateKey, _claims JWT.Claims) (string, error) {
	// alg/signing-method
	alg_name := "ES384"
	method := JWT.GetSigningMethod(alg_name)
	if method == nil {
		return "", fmt.Errorf("can not find signing method: %v", alg_name)
	}

	// create a new token
	token := JWT.NewWithClaims(method, _claims)

	// write header (map string->interface{})
	token.Header["header-1"] = 2
	token.Header["header-2"] = "hi2"

	// sign
	out, err := token.SignedString(_key)
	check(err)
	return out, nil
}

// verify
func verify_jwt(_key *ecdsa.PublicKey, _token_data []byte) (*JWT.Token, error) {
	// trim whitespace
	token_str := regexp.MustCompile(`\s*$`).ReplaceAll(_token_data, []byte{})
	fmt.Fprintf(os.Stderr, "Token len: %v bytes\n", len(token_str))

	// parse
	token, err := JWT.Parse(string(token_str), func(t *JWT.Token) (interface{}, error) { return _key, nil })
	if token != nil {
		fmt.Fprintf(os.Stderr, "Header:\n  %v\n\n", token.Header)
		fmt.Fprintf(os.Stderr, "Claims:\n  %v\n\n", token.Claims)
	}
	check(err)
	return token, nil
}

// Print a json object in accordance with the prophecy (or the command line options)
func printJSON(j interface{}) error {
	var out []byte
	var err error

	out, err = json.MarshalIndent(j, "", "    ")
	if err == nil {
		fmt.Println(string(out))
	}
	return err
}

/*
// showToken pretty-prints the token on the command line.
func showToken() error {
  // get the token
  tokData, err := loadData(*flagShow)
  if err != nil {
    return fmt.Errorf("Couldn't read token: %v", err)
  }

  // trim possible whitespace from token
  tokData = regexp.MustCompile(`\s*$`).ReplaceAll(tokData, []byte{})
  if *flagDebug {
    fmt.Fprintf(os.Stderr, "Token len: %v bytes\n", len(tokData))
  }

  token, err := JWT.Parse(string(tokData), nil)
  if token == nil {
    return fmt.Errorf("malformed token: %v", err)
  }

  // Print the token details
  fmt.Println("Header:")
  if err := printJSON(token.Header); err != nil {
    return fmt.Errorf("Failed to output header: %v", err)
  }

  fmt.Println("Claims:")
  if err := printJSON(token.Claims); err != nil {
    return fmt.Errorf("Failed to output claims: %v", err)
  }

  return nil
}
*/
