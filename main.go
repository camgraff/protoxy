package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/camgraff/proto-proxy/hello"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("hit")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		fmt.Println("Body: ", string(body))
		fmt.Println("Hello", hello.Hello{Hi: "hi there", Number: 12})

		parser := protoparse.Parser{}
		descriptors, err := parser.ParseFiles("hello.proto")
		if err != nil {
			log.Printf("Error parsing proto file: %v", err)
			return
		}
		fmt.Println(descriptors[0])
		msg := descriptors[0].FindMessage("hello.Hello")
		if msg == nil {
			log.Printf("Unable to find message: hello.Hello")
			return
		}
		fmt.Println(msg)
		fac := dynamic.NewMessageFactoryWithDefaults()
		helloMsg := fac.NewMessage(msg)
		helloMsgV2 := proto.MessageV2(helloMsg)
		fmt.Println(helloMsg)
		err = protojson.Unmarshal(body, helloMsgV2)
		if err != nil {
			log.Printf("Unable to unmarshal into json: %v", err)
		}
		fmt.Println(helloMsgV2)

	})
	log.Fatal(http.ListenAndServe(":7777", nil))
}
