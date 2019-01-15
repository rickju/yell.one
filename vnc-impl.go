package main

import (
  "context"
  "fmt"
  "io"
  "log"
  "net"
  "time"
  GRPC "google.golang.org/grpc"
   API "github.com/rickju/yell.one/build" // vnc.pb.go
)

type c_evt struct {
  // for sssn evt
  name   string
  p_agnt *c_agnt
  p_sssn *c_sssn

  // for reading grpc rqst
  rqst     *API.RqstVnc
  read_err error
}

type c_agnt struct {
  uuid string

  sssn_uuid string
  sssn_scrt string
  nice_cnfd string

  ichn      chan c_evt // input
  ochn_done chan int   // done

  p_grpc_stream API.VNC_BidiStrmServer
}

type c_sssn struct {
  uuid   string
  secret string

  agnt_list map[string]*c_agnt // uuid -> agnt
  ichn      chan c_evt         // input
  ochn_done chan int           // done
}

type c_mngr struct {
  sssn_list map[string]*c_sssn // uuid -> sssn
  ichn      chan c_evt         // input
  ochn_done chan int           // done
}

// agnt impl
// ----------------------------------
func create_agnt(_stream API.VNC_BidiStrmServer) *c_agnt {
  p_agnt := &c_agnt{p_grpc_stream: _stream}
  p_agnt.ichn = make(chan c_evt)
  p_agnt.ochn_done = make(chan int)
  return p_agnt
}

// lives in a goroutine, can be called by anybody, like "go p->on_evt(evt)"
func (_this *c_agnt) on_evt(_evt c_evt) {
  select {
  case _this.ichn <- _evt: // write
  case <-_this.ochn_done: // done
  }
}

func (_this *c_agnt) dojob_loop() {
  b_joined := false
  b_left := false
  b_reading := false
  b_read_done := false
  b_done := false

  // init
  _this.ichn = make(chan c_evt)
  _this.ochn_done = make(chan int)

  // forever loop
  for !b_done {
    // 500ms timeout ctx
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()

    // read in a goroutine, and fire the rqst it got by calling agnt::on_evt
    if !b_reading && !b_read_done {
      b_reading = true
      go func() {
        // grpc read
        rqst, err := _this.p_grpc_stream.Recv()
        // fire (even when recv failed, so b_reading can be resetted)
        evt := c_evt{read_err: err, rqst: rqst}
        _this.on_evt(evt)
      }()
    }

    select {
    case evt := <-_this.ichn:
      if evt.rqst != nil || evt.read_err != nil {
        b_reading = false
        if evt.read_err == io.EOF {
          // got read EOF
          log.Printf("agnt(%v:%v): <<<< grpc read EOF", _this.sssn_uuid, _this.uuid)
          b_read_done = true
          b_done = true
        } else if evt.read_err != nil {
          // got read error
          log.Printf("agnt(%v:%v): <<<< grpc read error: %v", _this.sssn_uuid, _this.uuid, evt.read_err)
          b_read_done = true
          b_done = true
        } else {
          // got a rqst
          // log.Printf("agnt(%v: %v): got a grpc rqst. %v  %v", 
          //            _this.sssn_uuid, _this.uuid, evt.rqst.AgntUuid, evt.rqst.Name)
          if evt.rqst.Name == "join" { // grpc I join
            b_joined = true
            _this.uuid = evt.rqst.AgntUuid
            _this.sssn_uuid = evt.rqst.SssnUuid
            _this.sssn_scrt = evt.rqst.SssnScrt
            _this.nice_cnfd = evt.rqst.NiceCnfd
            log.Printf("agnt(%v:%v): c2s >>>> %v %v", _this.sssn_uuid, _this.uuid, _this.uuid, evt.rqst.Name)
            evt := c_evt{
                name: "join",
              p_agnt: _this}
            go g_top_mngr.on_evt(evt)
          } else if evt.rqst.Name == "leave" { // grpc I leave
            b_left = true
            log.Printf("agnt(%v:%v): c2s >>>> %v %v", _this.sssn_uuid, _this.uuid, _this.uuid, evt.rqst.Name)
            evt := c_evt{
              name:   "leave",
              p_agnt: _this}
            go g_top_mngr.on_evt(evt)
          } else {
            // panic
            panic(fmt.Sprintf("agnt(%v:%v): unknown grpc rqst: %v", _this.sssn_uuid, _this.uuid, evt.rqst))
          }
        }
      } else if evt.p_agnt != nil && (evt.name == "join" || evt.name == "leave") {
        // got a peer evt
        log.Printf("agnt(%v:%v): s2c <<<< %v %v", _this.sssn_uuid, _this.uuid, evt.p_agnt.uuid, evt.name)
        // grpc write
        p_agnt := evt.p_agnt
        rply := API.RplyVnc {
              Name: evt.name,
          AgntUuid: p_agnt.uuid,
          SssnUuid: p_agnt.sssn_uuid,
          SssnScrt: p_agnt.sssn_scrt,
          NiceCnfd: p_agnt.nice_cnfd}
        if err := _this.p_grpc_stream.Send(&rply); err != nil {
          // grpc err or disconnected
          log.Printf("agnt(%v:%v): s2c <<<< grpc error: %v", _this.sssn_uuid, _this.uuid, err)
          b_done = true
        }
      } else {
        // panic
        panic(fmt.Sprintf("agnt(%v:%v): s2c <<<< unknown evt: %v", _this.sssn_uuid, _this.uuid, evt.name))
      }

    case <-ctx.Done(): // timeout
      // XXX send a grpc ping/pong, just to make sure grpc conn is alive?
      break

    case <-g_top_mngr.ochn_done: // app done
      log.Printf("agnt(%v:%v): s2c <<<< app done", _this.sssn_uuid, _this.uuid)
      b_done = true
    } // slct
  } // for

  if b_done {
    close(_this.ochn_done)

    if b_joined && !b_left {
      log.Printf("agnt(%v:%v): s2c >>>> %v leave", _this.sssn_uuid, _this.uuid, _this.uuid)
      evt := c_evt{
          name: "leave",
        p_agnt: _this}
      go g_top_mngr.on_evt(evt)
    }

    log.Printf("agnt(%v:%v): done", _this.sssn_uuid, _this.uuid)
  }
}

func (_this c_agnt) print () {
  log.Printf("agnt(%v:%v): ", _this.sssn_uuid, _this.uuid, )
}

// sssn impl
// ----------------------------------
func create_sssn(_uuid string, _secret string) *c_sssn {
  p_sssn := &c_sssn{uuid: _uuid, secret: _secret}
  p_sssn.agnt_list = make(map[string]*c_agnt)
  p_sssn.ichn = make(chan c_evt)
  p_sssn.ochn_done = make(chan int)
  return p_sssn
}

func (_this c_sssn) on_evt(_evt c_evt) { // lives in a goroutine, can be called by anybody
  select {
  case _this.ichn <- _evt: // write
  case <-_this.ochn_done: // done
  }
}

func (_this *c_sssn) dojob_loop() {
  b_done := false
  // for ever loop
  for !b_done {
    select {
    case evt := <-_this.ichn:
      if (evt.p_agnt != nil) {
        log.Printf("sssn(%v): got evt %v. alice_%v", _this.uuid, evt.name, evt.p_agnt.uuid)
      } else {
        log.Printf("sssn(%v): got evt %v. agnt: nil", _this.uuid, evt.name)
      }
      if evt.p_agnt != nil && evt.name == "join" {
        // send existing peers list to new agnt
        for _, p_agnt := range _this.agnt_list {
          tmp_evt := c_evt{
              name: "join",
            p_agnt: p_agnt}
          go evt.p_agnt.on_evt(tmp_evt)
        }
        // insert
        _this.agnt_list[evt.p_agnt.uuid] = evt.p_agnt
        // broadcast peers the newly joined
        for _, p_agnt := range _this.agnt_list {
          go p_agnt.on_evt(evt)
        }
      } else if evt.p_agnt != nil && evt.name == "leave" {
        delete(_this.agnt_list, evt.p_agnt.uuid)
        if len(_this.agnt_list) == 0 {
          // empty sssn
          log.Printf("sssn(%v): empty.", _this.uuid)
          evt := c_evt{ name: "sssn-empty", p_sssn: _this}
          go g_top_mngr.on_evt(evt)
        } else {
          // broadcast
          for _, p_agnt := range _this.agnt_list {
            go p_agnt.on_evt(evt)
          }
        }
      } else if evt.p_sssn != nil && evt.name == "sssn-done" {
        // done
        log.Printf("sssn(%v): done.", _this.uuid)
        if (evt.p_sssn != _this) {
          panic ("sssn(%v): invld evt sssn-done")
        }
        b_done = true
      } else {
        panic(fmt.Sprintf("sssn(%v): unknown evt: %v", _this.uuid, evt))
      }

    case <-g_top_mngr.ochn_done:
      // mngr done
      log.Printf("sssn(%v): got app done", _this.uuid)
      b_done = true
    }
  }

  if b_done {
    close(_this.ochn_done)
    log.Printf("sssn(%v): done", _this.uuid)
  }
}

func (_this c_sssn) print () {
  log.Printf("sssn (%v): peer num: %v", _this.uuid, len(_this.agnt_list))
  for _, p_agnt := range _this.agnt_list {
    p_agnt.print()
  }
}

// top mngr impl
// -----------------------------
func create_mngr() *c_mngr {
  p_mngr := &c_mngr{}
  p_mngr.sssn_list = make(map[string]*c_sssn)
  p_mngr.ichn = make(chan c_evt)
  p_mngr.ochn_done = make(chan int)
  return p_mngr
}

// lives in a goroutine, can be called by anybody
func (_this c_mngr) on_evt(_evt c_evt) {
  select {
  case _this.ichn <- _evt: // write
  case <-_this.ochn_done: // done
  }
}

func (_this c_mngr) dojob_loop() {
  // 
  // printer := printer()

  // for ever loop
  for b_app_done := false; !b_app_done; {
    // 500ms timeout ctx
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()

    select {
    case evt := <-_this.ichn:
      if nil != evt.p_agnt {
        log.Printf("mngr: got an evt: %v, agnt: %v\n", evt.name, evt.p_agnt.uuid)
      } else if nil != evt.p_sssn {
        log.Printf("mngr: got an evt: %v, agnt: nil, sssn: %v\n", evt.name, evt.p_sssn.uuid)
      } else {
        log.Printf("mngr: got an evt: %v, agnt: nil, sssn: nil\n", evt.name)
      }
      if evt.p_agnt != nil && (evt.name == "join" || evt.name == "leave") {
        uuid := evt.p_agnt.sssn_uuid
        log.Printf("agnt uuid: %v", uuid)
        // search list
         p_sssn := _this.sssn_list[uuid]
        if nil == p_sssn {
          // create new sssn if !exist
          log.Printf("mngr:  create sssn. %v", uuid)
          p_sssn = create_sssn(uuid, evt.p_agnt.sssn_scrt)
          _this.sssn_list[uuid] = p_sssn
          // forever sssn loop
          go p_sssn.dojob_loop()
        }
        // XXX would crash if reading sssn::agnt_list XXX
        // log.Printf("p_sssn: %v", p_sssn.uuid)
        // fire
        go p_sssn.on_evt(evt)
      }  else if (evt.name == "sssn-empty") {
        if (evt.p_sssn == nil) {
          panic(fmt.Sprintf("mngr: sssn-empty evt: p_sssn is nil"))
        }
        /* // maybe thing has changed. (sb joined somehow)
        if (len(evt.p_sssn.agnt_list)>0) {
          panic(fmt.Sprintf("mngr: sssn-empty evt: p_sssn.agnt_list is not empty "))
        } */
        if (len(evt.p_sssn.agnt_list) == 0) {
          delete(_this.sssn_list, evt.p_sssn.uuid)
          evt_done := c_evt{p_sssn: evt.p_sssn, name: "sssn-done"}
          go evt.p_sssn.on_evt(evt_done)
        }
      } else {
        // XXX
        // log.Printf("mngr: unknown evt: %v", evt)
        panic(fmt.Sprintf("mngr: unknown evt: %v", evt))
      }
      break

    case <-_this.ochn_done: // done
      log.Printf("mngr: got app done")
      b_app_done = true
      break

    case <-ctx.Done(): // timeout
      // XXX send a grpc ping/pong, just to make sure grpc conn is alive?
      break
    }  // slct

    // monitor/print health summary
    // XXX access sssn::agnt_list could cause a race condition
    // printer(_this)
  } // for
}

// print health info
func (_this c_mngr) print() {
  log.Printf("mngr: active sssn num: %v", len(_this.sssn_list))

  for _, p_sssn := range _this.sssn_list {
    p_sssn.print()
  }
}

// a closure. to print every 10 seconds
func printer() func(c_mngr) {
  start := time.Now()
  return func(_mngr c_mngr) {
    now := time.Now()
    elapse := now.Sub(start)  
    if elapse > 10*time.Second {
      _mngr.print()
      start = now
    }
  }
}

// entry
// ------------------------
var g_top_mngr *c_mngr

// forever looping to keep the call alive
func proc_bidi_strm_call(_stream API.VNC_BidiStrmServer) {
  // create new agnt
  p_agnt := create_agnt(_stream)

  // forever. this only returns when grpc disconn-ted/err or app exit
  p_agnt.dojob_loop()
}

func run_grpc_server() error {
  g_top_mngr = create_mngr()
  // forever loop
  go g_top_mngr.dojob_loop()

  // grpc server
  lis, err := net.Listen("tcp", port)
  if err != nil {
    log.Fatalf("failed to listen: %v", err)
  }
  grpc := GRPC.NewServer()

  // forever block
  // run grpc service "vnc".
  API.RegisterVNCServer(grpc, &vnc{})
  if err := grpc.Serve(lis); err != nil {
    log.Fatalf("failed to serve: %v", err)
  }
  return nil
}

/* XXX
func (_this c_mngr) finalize_grpc_server() {
  close(_this.ochn_done)
}
*/
