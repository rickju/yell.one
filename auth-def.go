package main

import (
  "github.com/google/uuid"
  // "fmt"
)

// backend API
// -----------------------
//
// - targets: 
// 
//   user
//   client(app)
//   vnc-session
//   conn
//
// - create/destroy/join/leave vnc session
//
//   CONN get_conn_info(SESSION, peer CLIENT)
//
// - how vnc session works:
//
// - vnc server: 
//
//    allow/revoke(id, secret) 
//    start/stop listen
//
// - vnc client:
//
//    get_vnc_info(id, secret)
//    conn (ip, ...)
//
// - jwt: ?
//
// - webRTC/turn/ice conn: ?
//
// user
type USER struct {
     id uuid.UUID
  email string
   nick string

   // client uuid (current, history)
   // country
   // lang
   // verified email
   // paid status, pay history
   // create time
}

// client (app)
type CLIENT struct {
  id uuid.UUID

  // os info(os, version, cpu)
  // install/uninstall log.    (idea: use jwt to save at client end?)
  // start/stop log
  // external ip log
}

// vnc session
type VNC_SESSION struct {
      id uuid.UUID
  secret string
   owner uuid.UUID  // user uuid
  // expire time
  // member list
}

// conn: desc a peer client instance's conn info
type CONN struct {
  owner uuid.UUID  // owner client id
  // ip addr
  // nat info ....
}

// access token: in jwt
type ACS_TOKEN struct {
  user uuid.UUID  // user id
   app uuid.UUID  // app
   // expire time
}

func allow_vnc(id string, secret string) error {
  return nil
}

// revoke all if empty id 
func revoke_vnc(id string) error {
  return nil
}

func start_listen() error {
  return nil
}

func stop_listen() error {
  return nil
}

func get_vnc_info(id string, secret string)  error {
  return nil
}



