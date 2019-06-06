package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dtn7/dtn7-go/core"
	"github.com/ugorji/go/codec"
)

func buildUrl(host, action string) string {
	if strings.HasSuffix(host, "/") {
		return fmt.Sprintf("%s%s/", host, action)
	} else {
		return fmt.Sprintf("%s/%s/", host, action)
	}
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

	if resp.StatusCode != 200 {
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

func fetchRequest(host string) error {
	resp, err := http.Get(buildUrl(host, "fetch"))
	if err != nil {
		return err
	}

	json, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("%s", string(json))
	return nil
}

func showHelp() {
	fmt.Printf("dtncat [send|fetch|help] ...\n\n")
	fmt.Printf("dtncat send REST-API ENDPOINT-ID\n")
	fmt.Printf("  sends data from stdin through the given REST-API to the endpoint\n\n")
	fmt.Printf("dtncat fetch REST-API\n")
	fmt.Printf("  fetches all bundles from the given REST-API\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  dtncat send  \"http://127.0.0.1:8080/\" \"dtn:alpha\" <<< \"hello world\"\n")
	fmt.Printf("  dtncat fetch \"http://127.0.0.1:8080/\"\n")
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		showHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "send":
		if len(args) != 3 {
			fmt.Printf("Amount of parameters is wrong.\n\n")
			showHelp()
			os.Exit(1)
		}

		payload, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Failed to read data from stdin: %v", err)
			os.Exit(1)
		}

		if err = sendRequest(args[1], args[2], payload); err != nil {
			fmt.Printf("Sending data failed: %v", err)
			os.Exit(1)
		}

	case "fetch":
		if len(args) != 2 {
			fmt.Printf("Amount of parameters is wrong.\n\n")
			showHelp()
			os.Exit(1)
		}

		if err := fetchRequest(args[1]); err != nil {
			fmt.Printf("Fetching data failed: %v", err)
			os.Exit(1)
		}

	case "help", "--help", "-h":
		showHelp()

	default:
		fmt.Printf("Unknown option: %s\n\n", args[0])
		showHelp()
		os.Exit(1)
	}
}
