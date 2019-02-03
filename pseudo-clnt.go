//
// rick: based on grpc-go example routeguide/route_guide.proto.
//
package main

import (
  "context"
  "fmt"
  "google.golang.org/grpc"
  "io"
  "log"
  "strconv"
  "time"

  //API "./vendor" // vnc.pb.go
  // API "woodboard" // vnc.pb.go
  // API "github.com/rickju/yell.one/build" // vnc.pb.go
  API "github.com/rickju/yell.one/build" // vnc.pb.go
)

//
// design: how to test bknd: join/leave
// ------------------------------------
//
//  1. create a list of pseudo client. number: peer_count
//  2. each client connect to bknd independently
//  3. each client has a id/agnt-uuid [0, peer_count-1]
//  4. client N will leave only when:
//
//     a. after all clients [N+1, Max] leave. client MAX will leave 10 seconds after it joined.
//     b. after all peers joined
//
//  5. all clients will check if it got join evt from all peers
//  6. client-N check if it got leave evt from client [N+1, Max]
//

const g_sssn_count = 1000
const g_peer_count = 10

type c_pseudo_client struct {
  id     int
  b_done bool

  agnt_uuid string
  sssn_uuid string
  sssn_scrt string
  nice_cnfd string
}

type c_pseudo_sssn struct {
  id     int
  b_done bool

  sssn_uuid string
  sssn_scrt string
}

// clnt impl
// -----------
func (_this *c_pseudo_client) init(_id int, _agnt_uuid string, _sssn_uuid string, _sssn_scrt string, _nice_cnfd string) {
  _this.b_done = false
  _this.id = _id
  _this.agnt_uuid = _agnt_uuid
  _this.sssn_uuid = _sssn_uuid
  _this.sssn_scrt = _sssn_scrt
  _this.nice_cnfd = _nice_cnfd
}

func (_this *c_pseudo_client) test() {
  // grpc conn
  var opts []grpc.DialOption
  /*
      // using tls
      if *tls {
        if *caFile == "" {
          *caFile = testdata.Path("ca.pem")
        }
        creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
        if err != nil {
          log.Fatalf("failed to create TLS credentials %v", err)
        }
        opts = append(opts, grpc.WithTransportCredentials(creds))
      } else {
        opts = append(opts, grpc.WithInsecure())
      }
  */
  opts = append(opts, grpc.WithInsecure())
  opts = append(opts, grpc.WithBlock())

  // grpc conn
  serverAddr := "127.0.0.1:50051"
  conn, err := grpc.Dial(serverAddr, opts...)
  if err != nil {
    log.Fatalf("fail to dial: %v", err)
  }
  defer conn.Close()
  client := API.NewVNCClient(conn)

  // peer list: make sure got a join from each of them
  var peer_join [g_peer_count]bool  // a list of joined peer-test-id
  var peer_leave [g_peer_count]bool // a list of left peer-test-id
  for i := 0; i < g_peer_count; i++ {
    peer_join[i] = false
    peer_leave[i] = false
  }

  // init
  stream, err := client.BidiStrm(context.Background())
  if err != nil {
    log.Fatalf("%v.BidiStrm(_) = _, %v", client, err)
  }

  // write "join"
  // log.Printf("agnt(%v:%v): >>>> %v join", _this.sssn_uuid, _this.id, _this.id)
  rqst := API.RqstVnc {
    Name:     "join",
    AgntUuid: _this.agnt_uuid,
    SssnUuid: _this.sssn_uuid,
    SssnScrt: _this.sssn_scrt}
  if err := stream.Send(&rqst); err != nil {
    log.Fatalf("agnt(%v:%v): failed to send a rqst: %v", _this.sssn_uuid, _this.id, err)
  }

  b_should_leave := false
  for !b_should_leave {
    // check
    b_should_leave = true
    // peer[0...Max] joined
    for i := 0; i < g_peer_count; i++ {
      if !peer_join[i] {
        b_should_leave = false
        break
      }
    }
    // peer[my-id+1...Max] left
    for i := 1 + _this.id; i < g_peer_count; i++ {
      if !peer_leave[i] {
        b_should_leave = false
        break
      }
    }
    if b_should_leave {
      break
    }

    // read (block forever)
    // log.Printf("agnt(%v:%v): reading rply....", _this.sssn_uuid, _this.id)
    rply, err := stream.Recv()
    if err == io.EOF { // read done.
      log.Printf("agnt(%v:%v): <<<< read EOF", _this.sssn_uuid, _this.id)
      return
    } else if err != nil { // read err
      log.Fatalf("agnt(%v:%v): failed to receive reply: %v", _this.sssn_uuid, _this.id, err)
      panic("read error")
      return
    } else { // got a rply
      // log.Printf("agnt(%v:%v): <<<< %v %v", _this.sssn_uuid, _this.agnt_uuid, rply.AgntUuid, rply.Name)
      agnt_uuid, err := strconv.Atoi(rply.AgntUuid)
      if err != nil {
        panic(fmt.Sprintf("not a int: %v", rply.AgntUuid))
      }
      if rply.Name == "join" {
        if peer_join[agnt_uuid] {
          panic("already joined")
        }
        peer_join[agnt_uuid] = true
      } else if rply.Name == "leave" {
        if peer_leave[agnt_uuid] {
          panic("already left")
        }
        peer_leave[agnt_uuid] = true
      }
    }
  } // for

  // write: leave
  log.Printf("agnt(%v:%v): >>>> %v leave", _this.sssn_uuid, _this.id, _this.id)
  rqst_leave := API.RqstVnc{ Name:"leave", AgntUuid: _this.agnt_uuid}
  if err := stream.Send(&rqst_leave); err != nil {
    log.Fatalf("agnt(%v:%v): failed to send leave rqst: %v", _this.sssn_uuid, _this.id, err)
    panic(fmt.Sprintf("agnt(%v:%v): failed to send leave rqst. agnt: %v", _this.sssn_uuid, _this.id, _this))
  }

  // XXX if close now, sometimes, bknd can NOT get leave rqst, just get a read error.
  /*
    log.Printf("%v:%v: sleep", _this.sssn_uuid, _this.id)
    time.Sleep(10 * time.Second)
  */
  log.Printf("agnt(%v:%v): closing grpc stream", _this.sssn_uuid, _this.id)
  stream.CloseSend()
  _this.b_done = true
  log.Printf("agnt(%v:%v): done. b_done: %v, sssn_scrt: %v", _this.sssn_uuid, _this.id, _this.b_done, _this.sssn_scrt)
}

// sssn impl
// ----------
func (_this *c_pseudo_sssn) init(_id int, _sssn_uuid string, _sssn_scrt string) {
  _this.b_done = false
  _this.id = _id
  _this.sssn_uuid = _sssn_uuid
  _this.sssn_uuid = _sssn_uuid
  _this.sssn_scrt = _sssn_scrt
}

func (_this *c_pseudo_sssn) test() {
  nice_cnfd := "nice-cnfd-111"

  list_clnt := [g_peer_count]c_pseudo_client{}
  for i := 0; i < g_peer_count; i++ {
    clnt := &list_clnt[i]
    agnt_uuid := strconv.Itoa(i)
    clnt.init(i, agnt_uuid, _this.sssn_uuid, _this.sssn_scrt, nice_cnfd)
    go clnt.test()
  }

  // sleep until all clnt done
  for {
    b_done := true
    for i := 0; i < g_peer_count; i++ {
      if !list_clnt[i].b_done {
        b_done = false
      }
    }
    if !b_done {
      time.Sleep(100 * time.Millisecond)
      // log.Printf("sssn(%v): alive", _this.id)
    } else {
      break
    }
  }
  _this.b_done = true
  log.Printf("sssn(%v): done", _this.id)
}

// entry
// -----
func run_pseudo_clnt() {
  // start
  list_sssn := [g_sssn_count]c_pseudo_sssn{}
  for i := 0; i < g_sssn_count; i++ {
    sssn := &list_sssn[i]
    sssn_uuid := strconv.Itoa(i)
    sssn_scrt := "sssn-scrt-111"
    sssn.init(i, sssn_uuid, sssn_scrt)
    go sssn.test()
  }

  // sleep until all sssn done
  for {
    b_done := true
    for i := 0; i < g_sssn_count; i++ {
      if !list_sssn[i].b_done {
        b_done = false
      }
    }
    if !b_done {
      time.Sleep(100 * time.Millisecond)
    } else {
      break
    }
  }
  log.Printf("test: done ------------")
}
