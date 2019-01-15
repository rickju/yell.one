package main

import (
  "io"
  "log"
  "time"
  API "github.com/rickju/yell.one/build" // vnc.pb.go
)

// listen port
const (
  // port = ":50051"
  port = ":80"
)

// data storage for VNC
type vnc struct {
  SvrId        string
  SvrSecret    string
  SvrNiceCnfd  string
  ClntNiceCnfd string
}

// grpc bidirection streaming
func (s *vnc) BidiStrm(_stream API.VNC_BidiStrmServer) error {
  // log.Printf("BidiStrm called --------")
  proc_bidi_strm_call(_stream)
  return nil
}

// just for test
// test grpc bidirection streaming
// pls refer to example in github.com/grpc-go/git/examples/route_guide/server
func (s *vnc) TestBidiStrm(_stream API.VNC_TestBidiStrmServer) error {
  log.Printf("TestBidiStrm rqst Id")

  // for test -------------------------
  i := 0
  for {
    rqst, err := _stream.Recv()
    log.Printf("got requst: >>>> : %v", rqst)
    if err == io.EOF {
      return nil
    }
    if err != nil {
      return err
    }

    log.Printf("            <<<<<  :   id: %v", i)
    rply := API.RplyTestBidiStrm{Id: string(i), Evt: "3333", Rtn: 200}
    if err := _stream.Send(&rply); err != nil {
      return err
    }
    i = i + 1

    // MUST NOT return
    // once return, whole context got invalid, and grpc call got finished.
    // sleep forever
    time.Sleep(100 * time.Millisecond)
  }
  return nil
}
