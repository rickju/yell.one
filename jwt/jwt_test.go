package jwt

import (
"testing"
JWT "github.com/dgrijalva/jwt-go"
"fmt"
"time"
)

func TestJwt(t *testing.T ){
//func TestJwt() {
	// gen pem
	TestEcdsa()

	//
	duration := time.Hour * 24 * 30 // 30 days
	expires := time.Now().Add(duration).Unix()

	// claims
	type MyCustomClaims struct {
		Foo      string `json:"foo"`
		ClientId string `json:"clientid"` // client/app uuid
		Arch     string `json:"arch"`     // arch (ia32/x64)
		Os       string `json:"os"`       // os type (win/osx/linux/...)
		OsVerion string `json:"osv"`      // os version
		HostId   string `json:"hostid"`   // os host UUID
		HDSerial string `json:"hds"`      // hard drive serial

		JWT.StandardClaims
	}

	claims := MyCustomClaims{
		"bar",
		"cid-343",
		"x64",
		"Win",
		"10.0.0.1563",
		"hid-343",
		"hds-343",
		JWT.StandardClaims{
			ExpiresAt: expires,
			Issuer:    "woodboard",
		},
	}

	// priv key
	priv_key, err := read_priv("./priv.pem")
	check(err)
	// pub key
	pub_key, err := read_pub("./pub.pem")
	check(err)

	// gen
	tkn, _ := gen_jwt(priv_key, claims)
	fmt.Printf("\n----- gen-ed token ----\n%v\n\n", tkn)

	// verify
	fmt.Println("\n---- verify ----: ")

	token, err := verify_jwt(pub_key, []byte(tkn))
	check(err)

	if !token.Valid {
		fmt.Printf("Error: Token is invalid")
	} else {
		fmt.Printf("  token verified")
	}
	printJSON(token.Claims)
	{
		claims2 := token.Claims.(JWT.MapClaims)
		expire2 := claims2["exp"].(float64)
		delta := time.Until(time.Unix(int64(expire2), 0))
		fmt.Printf("  %v to expire\n", delta)
	}
}
