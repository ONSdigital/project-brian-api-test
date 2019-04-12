package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

func main() {
	for _, filename := range []string{"ott", "bb", "berd", "ragv", "ukea", "sppi"} {
		fmt.Printf("capturing project-brian response for dataset: %s\n", filename)
		err := storeBaselineResponses(filename)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("finished project-brian responses")
}

func storeBaselineResponses(filename string) error {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename+".csdb")
	if err != nil {
		return err
	}

	f, err := os.Open(fmt.Sprintf("resources/inputs/%s.csdb", filename))
	if err != nil {
		return err
	}
	defer f.Close()

	copied, err := io.Copy(fileWriter, f)
	if err != nil {
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if fi.Size() != copied {
		return errors.New("number bytes copied to request did not match the file size")
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post("http://localhost:8083/Services/ConvertCSDB", contentType, body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	var temp []interface{}

	err = json.Unmarshal(respData, &temp)
	if err != nil {
		return err
	}

	pretty, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("resources/outputs/%s-csdb.json", filename), pretty, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
