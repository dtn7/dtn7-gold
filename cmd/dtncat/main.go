package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/geistesk/dtn7/core"
	"github.com/ugorji/go/codec"
)

func buildUrl(host, endpoint string) string {
	if strings.HasSuffix(host, "/") {
		return fmt.Sprintf("%s%s", host, endpoint)
	} else {
		return fmt.Sprintf("%s/%s", host, endpoint)
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

func main() {
	payload, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("Failed to read stdin: %v", err)
		return
	}

	if err = sendRequest("http://localhost:8081/", "dtn:host2", payload); err != nil {
		fmt.Printf("Sending failed: %v", err)
		return
	}
}
