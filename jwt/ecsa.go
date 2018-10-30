package jwt

import (
  "crypto/ecdsa"
  "crypto/elliptic"
  "crypto/rand"
  "crypto/x509"
  "crypto/sha256"
  "io/ioutil"
  "reflect"
  "fmt"
  PEM "encoding/pem"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func save_priv(_file_name string, _privateKey *ecdsa.PrivateKey) error {
  x509Encoded, _ := x509.MarshalECPrivateKey(_privateKey)
  pemEncoded := PEM.EncodeToMemory(&PEM.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

  err := ioutil.WriteFile(_file_name, pemEncoded, 0644)
  check(err)
  return err
}

func save_pub(_file_name string, _publicKey *ecdsa.PublicKey) error {
  x509EncodedPub, _ := x509.MarshalPKIXPublicKey(_publicKey)
  pemEncodedPub := PEM.EncodeToMemory(&PEM.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

  err := ioutil.WriteFile(_file_name, pemEncodedPub, 0644)
  check(err)
  return err
}

func read_priv(_file_name string) (*ecdsa.PrivateKey, error) {
  pem, err := ioutil.ReadFile(_file_name)
  check(err)

  block, _ := PEM.Decode([]byte(pem))
  x509Encoded := block.Bytes
  privateKey, _ := x509.ParseECPrivateKey(x509Encoded)
  return privateKey, nil
}

func read_pub(_file_name string) (*ecdsa.PublicKey, error) {
  pem, err := ioutil.ReadFile(_file_name)
  check(err)

  blockPub, _ := PEM.Decode([]byte(pem))
  x509EncodedPub := blockPub.Bytes
  genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
  publicKey := genericPublicKey.(*ecdsa.PublicKey)
  return publicKey, nil
}

// test
func TestEcdsa() {
  //
  privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
  publicKey := &privateKey.PublicKey
  if err != nil {
    panic(err)
  }

  save_priv("priv.pem", privateKey)
  priv2,err := read_priv("priv.pem")
  if !reflect.DeepEqual(privateKey, priv2) {
    fmt.Println("Private keys do NOT match.")
  }

  save_pub("pub.pem", publicKey)
  pub2,err := read_pub("pub.pem")
  if !reflect.DeepEqual(publicKey, pub2) {
    fmt.Println("Public keys do NOT match.")
  }

  //
  msg := "hello, world"
  hash := sha256.Sum256([]byte(msg))

  r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
  if err != nil {
    panic(err)
  }
  fmt.Printf("signature: (0x%x, 0x%x)\n", r, s)

  valid := ecdsa.Verify(&priv2.PublicKey, hash[:], r, s)
  fmt.Println("signature verified:", valid)
}

