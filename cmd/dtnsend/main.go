package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/dtn7/dtn7-go/core"
	"github.com/ugorji/go/codec"
)

func buildUrl(host, action string) string {
	u, _ := url.Parse(host)
	u.Path = path.Join(u.Path, action)
	return u.String() + "/"
}

func sendRequest(host, destination string, payload []byte) error {
	req := core.SimpleRESTRequest{
		Destination: destination,
		Payload:     base64.StdEncoding.EncodeToString(payload),
	}

	buff := new(bytes.Buffer)
	codec.NewEncoder(buff, new(codec.JsonHandle)).Encode(req)

	resp, err := http.Post(buildUrl(host, "send"), "application/json", buff)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Response's status code is %d != 200", resp.StatusCode)
	}

	var respData core.SimpleRESTRequestResponse
	if err := codec.NewDecoder(resp.Body, new(codec.JsonHandle)).Decode(&respData); err != nil {
		return err
	}

	if respData.Error != "" {
		return fmt.Errorf("JSON contains error: %v", respData.Error)
	}

	return nil
}

func showHelp() {
	fmt.Printf("dtnsend <EID>...\n\n")
	fmt.Printf("  sends data from stdin to the given endpoint\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  dtnsend  \"dtn://alpha/recv\" <<< \"hello world\"\n")
}

func main() {
	args := os.Args[1:]

	resthost := os.Getenv("DTN7RESTHOST")
	if resthost == "" {
		resthost = "http://127.0.0.1:8080"
	}

	if len(args) == 0 {
		showHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "help", "--help", "-h":
		showHelp()

	default:
		if len(args) != 1 {
			fmt.Printf("Amount of parameters is wrong.\n\n")
			showHelp()
			os.Exit(1)
		}

		payload, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Failed to read data from stdin: %v", err)
			os.Exit(1)
		}

		if err = sendRequest(resthost, args[0], payload); err != nil {
			fmt.Printf("Sending data failed: %v", err)
			os.Exit(1)
		}
	}
}
