package main

import (
  "github.com/google/uuid"
	"fmt"
)

// auth api
// -----------------------

// [deprecated]: no sign-up, send-verify-code only
// func signup (email string) error

//
func check_exist (email string) (user_uuid string){
  return "nil"
}

// gen a code, save code with expire, send email
func send_vrf_email (email string) error {
  return nil
}

// check code/expire in db
// gen token (using priv key)
func login (email string, vrf_code string) (acs_token string, err error) {
  return "nil", nil
}

// decode access token -> user_id. no need to read db.
func get_user_id (acs_token string) (user_uuid string, err error) {
  return "nil", nil
}

// decode access token -> user_id/email. no need to read db.
func get_user_info (access_token string) (email string, err error) {
  return "nil", nil
}

//
// check old token(sig/exp)
// gen&return new token
func refresh (access_token string) (acs_token string, err error) {
  return "nil", nil
}


//
func test_uuid() error {

  id,err := uuid.NewRandom()
  fmt.Printf("uuid: %v, err: %v\n", id, err)
  return nil
}

func test_auth() error {
  return nil
}



